/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runtimeprovider

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goplus/mod/modfile"
	"github.com/goplus/mod/modload"
	"github.com/goplus/mod/xgomod"
	"github.com/goplus/xgo/env"
	"github.com/goplus/xgo/x/xgoprojs"
	gomodfile "golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const runtimeGuardEnv = "XGO_RUNTIME_GUARD"

func runtimeGuard(projectDir, providerPackage string) string {
	return sha256Bytes([]byte(projectDir + "\x00" + providerPackage))
}

// Resolver owns one invocation's graph and host-build policies.
type Resolver struct {
	cwd        string
	policy     parsedFlags
	xgoVersion string
}

// NewResolver snapshots ambient GOFLAGS/GOWORK once for an invocation. Policy
// setup must not fall back to a different module graph: doing so could classify
// a workspace runtime target as legacy and incorrectly dispatch it to GenGo.
func NewResolver(ctx context.Context, cwd string, flags []string) (*Resolver, error) {
	cwd, err := canonicalExistingDir(cwd)
	if err != nil {
		return nil, err
	}
	policy, err := preparePolicies(ctx, cwd, flags)
	if err != nil {
		return nil, err
	}
	return &Resolver{cwd: cwd, policy: policy, xgoVersion: env.Version()}, nil
}

// BuildPolicy returns the validated runtime build policy. Callers must invoke
// it only after Resolve matched a runtime project.
func (r *Resolver) BuildPolicy() (BuildPolicy, error) {
	if err := r.policy.validateRuntime(); err != nil {
		return BuildPolicy{}, err
	}
	return r.policy.build, nil
}

// Resolve resolves a parsed XGo target without parsing or generating source.
func (r *Resolver) Resolve(ctx context.Context, target xgoprojs.Proj) (*Runtime, error) {
	var (
		kind             TargetKind
		original         string
		projectDir       string
		targetImportPath string
		expectedFile     string
		multiFile        bool
		unsupportedForm  string
		recursivePattern bool
		graph            *effectiveGraph
		graphWorkDir     string
	)
	switch target := target.(type) {
	case *xgoprojs.DirProj:
		kind, original = TargetDirectory, target.Dir
		candidate := target.Dir
		if hasRecursivePattern(candidate) {
			unsupportedForm = "directory pattern containing ..."
			recursivePattern = true
			candidate = trimRecursivePattern(candidate)
		}
		var err error
		projectDir, err = canonicalExistingDir(candidate)
		if err != nil {
			if (os.IsNotExist(errors.Unwrap(err)) || os.IsNotExist(err)) && r.policy.graph.Overlay != "" {
				kind, projectDir, expectedFile, graph, err = r.resolveOverlayLocalTarget(ctx, candidate, recursivePattern)
				if err != nil {
					return nil, err
				}
				graphWorkDir = r.cwd
				break
			}
			if os.IsNotExist(errors.Unwrap(err)) || os.IsNotExist(err) {
				return nil, ErrNotHandled
			}
			return nil, err
		}
		graphWorkDir = projectDir
	case *xgoprojs.FilesProj:
		if len(target.Files) == 0 {
			return nil, ErrNotHandled
		}
		kind, original = TargetFile, target.Files[0]
		multiFile = len(target.Files) != 1
		file, err := canonicalExistingFile(target.Files[0])
		if err != nil {
			return nil, ErrNotHandled
		}
		expectedFile = file
		projectDir = filepath.Dir(file)
		graphWorkDir = projectDir
	case *xgoprojs.PkgPathProj:
		kind, original, targetImportPath = TargetPackage, target.Path, target.Path
		lookupPath := target.Path
		if strings.Contains(lookupPath, "@") {
			hasRuntime, err := r.versionedPackageHasRuntime(ctx, lookupPath)
			if err != nil {
				return nil, err
			}
			if !hasRuntime {
				return nil, ErrNotHandled
			}
			return nil, fmt.Errorf("runtime provider v1 does not support package target containing @version")
		}
		if hasRecursivePattern(lookupPath) {
			unsupportedForm = "package pattern containing ..."
			recursivePattern = true
			lookupPath = trimRecursivePattern(lookupPath)
		}
		hasOverlay := r.policy.graph.Overlay != ""
		var (
			preflightModule modload.Module
			preflightGoMod  string
			callerHasClass  bool
			vendor          bool
			preflightErr    error
		)
		if !hasOverlay {
			preflightModule, preflightGoMod, callerHasClass, vendor, preflightErr = r.preflightClassMetadataDetails(ctx, r.cwd)
			if preflightErr != nil && !errors.Is(preflightErr, errNoGoModule) {
				return nil, preflightErr
			}
			if vendor {
				// Outside a workspace, an ordinary module with no class metadata
				// cannot select a runtime provider. Avoid asking `go list -m all`,
				// which is unsupported in vendor mode, and preserve its legacy path.
				if !callerHasClass && r.policy.graph.GoWork == "off" {
					return nil, ErrNotHandled
				}
				hasRuntime, probeErr := r.probeVendorPackage(ctx, preflightGoMod, preflightModule, lookupPath, recursivePattern)
				if probeErr != nil {
					return nil, probeErr
				}
				if hasRuntime {
					return nil, vendorUnsupportedError(string(r.policy.graph.ModMode))
				}
				return nil, ErrNotHandled
			}
		}
		callerGraph, err := loadEffectiveGraph(ctx, r.cwd, r.policy.graph)
		if err != nil {
			if errors.Is(err, errNoGoModule) && !callerHasClass && !hasOverlay {
				return nil, ErrNotHandled
			}
			return nil, err
		}
		var targetModule ResolvedModule
		projectDir, targetModule, err = resolvePackageDirectory(ctx, callerGraph, lookupPath, r.cwd, r.policy.graph)
		if err != nil {
			if callerHasClass || hasOverlay {
				return nil, err
			}
			return nil, ErrNotHandled
		}
		// A package target may use runtime metadata from its own main/workspace
		// module. A non-main dependency is eligible only when the caller's exact
		// target modfile snapshot marked that logical module as an XGo class
		// dependency. Do not let retargeting discover an unmarked dependency's
		// ambient gox.mod and expand the provider execution trust boundary.
		if !targetModule.Main && !classModuleMarked(callerGraph.ClassModules, targetModule.Selected.Path) {
			return nil, ErrNotHandled
		}
		graph, err = retargetEffectiveGraph(callerGraph, targetModule)
		if err != nil {
			return nil, err
		}
		graphWorkDir = r.cwd
	default:
		return nil, ErrNotHandled
	}
	graphPolicy := r.policy.graph
	graphPolicy.WorkDir = graphWorkDir

	if graph == nil {
		// An overlay changes the Go command's effective module graph and may
		// introduce class markers or runtime metadata that are absent from the
		// physical tree. Classify through that same graph before the physical
		// preflight; otherwise a runtime target can fall through to legacy.
		if overlay := graphPolicy.Overlay; overlay != "" {
			overlayGraph, graphErr := loadEffectiveGraph(ctx, projectDir, graphPolicy)
			if graphErr != nil {
				if errors.Is(graphErr, errNoGoModule) {
					return nil, ErrNotHandled
				}
				return nil, graphErr
			}
			hasRuntime, matchErr := overlayRuntimeProjectMatch(projectDir, overlayGraph, recursivePattern)
			if matchErr != nil {
				return nil, matchErr
			}
			if !hasRuntime {
				return nil, ErrNotHandled
			}
			return nil, unsupportedOverlayError(overlay)
		}
		preflightModule, _, hasClass, vendor, err := r.preflightClassMetadataDetails(ctx, projectDir)
		if err != nil {
			if errors.Is(err, errNoGoModule) {
				return nil, ErrNotHandled
			}
			return nil, err
		}
		if !hasClass {
			return nil, ErrNotHandled
		}
		if vendor {
			hasRuntime, probeErr := r.probeVendorProject(projectDir, preflightModule, recursivePattern)
			if probeErr != nil {
				return nil, probeErr
			}
			if !hasRuntime {
				return nil, ErrNotHandled
			}
			return nil, vendorUnsupportedError(string(r.policy.graph.ModMode))
		}
		graph, err = loadEffectiveGraph(ctx, projectDir, r.policy.graph)
		if err != nil {
			return nil, err
		}
	} else {
		if graph.files != nil && graph.files.hasOverlay() {
			hasRuntime, matchErr := overlayRuntimeProjectMatch(projectDir, graph, recursivePattern)
			if matchErr != nil {
				return nil, matchErr
			}
			if !hasRuntime {
				return nil, ErrNotHandled
			}
			return nil, unsupportedOverlayError(graphPolicy.Overlay)
		}
		hasClass, err := graphHasClassMetadata(graph)
		if err != nil {
			return nil, err
		}
		if !hasClass {
			return nil, ErrNotHandled
		}
	}
	module, _, err := loadResolvedClasses(graph)
	if err != nil {
		return nil, err
	}
	if recursivePattern {
		hasRuntime, err := patternContainsRuntimeProject(projectDir, module)
		if err != nil {
			return nil, err
		}
		if !hasRuntime {
			return nil, ErrNotHandled
		}
		return nil, fmt.Errorf("runtime provider v1 does not support %s", unsupportedForm)
	}
	projectFile, info, candidates, err := findProjectFile(projectDir, module)
	if err != nil {
		return nil, err
	}
	if info == nil || info.Project.Runtime == nil {
		return nil, ErrNotHandled
	}
	if unsupportedForm != "" {
		return nil, fmt.Errorf("runtime provider v1 does not support %s", unsupportedForm)
	}
	if candidates != 1 {
		return nil, fmt.Errorf("runtime project directory %q contains %d project files; exactly one is required", projectDir, candidates)
	}
	if multiFile {
		return nil, fmt.Errorf("runtime provider v1 does not support multiple source-file targets")
	}
	if expectedFile != "" {
		same, err := sameFile(expectedFile, projectFile)
		if err != nil || !same {
			return nil, fmt.Errorf("runtime file target %q is not the unique project file %q", original, projectFile)
		}
	}
	if info.Origin == nil {
		return nil, fmt.Errorf("runtime project %q has no module provenance", projectFile)
	}
	if err := checkRequiredXGo(info.RequiredXGo, r.xgoVersion); err != nil {
		return nil, err
	}
	project := info.Project
	if project.Runtime.Protocol != protocolV1 {
		return nil, fmt.Errorf("unsupported runtime provider protocol %q", project.Runtime.Protocol)
	}
	packDir, packIndex, err := validatePack(projectDir, project.Pack)
	if err != nil {
		return nil, err
	}
	origin := *info.Origin
	goxmod, err := declaringMetadata(origin, info.Declaration)
	if err != nil {
		return nil, err
	}
	if os.Getenv("XGO_RUNTIME") == "off" {
		return nil, ErrRuntimeDisabled
	}
	if os.Getenv(runtimeGuardEnv) != "" {
		return nil, ErrRuntimeRecursive
	}
	defaultName := defaultExecutableName(kind, projectDir, targetImportPath)
	return &Runtime{
		TargetKind:       kind,
		OriginalTarget:   original,
		TargetImportPath: targetImportPath,
		DefaultExecName:  defaultName,
		ProjectDir:       projectDir,
		ProjectFile:      projectFile,
		ModuleRoot:       graph.Target.Effective().Dir,
		ProviderPackage:  project.Runtime.Package,
		Origin:           origin,
		RequiredXGo:      info.RequiredXGo,
		Protocol:         project.Runtime.Protocol,
		ProjectExt:       project.Ext,
		ProjectFullExt:   project.FullExt,
		PackDir:          packDir,
		PackIndex:        packIndex,
		GoxMod:           goxmod.Path,
		GoxModSHA256:     goxmod.SHA256,
		Graph:            graphPolicy,
	}, nil
}

// versionedPackageHasRuntime classifies the requested module version in an
// isolated graph. It must not borrow metadata from the caller's selected build
// list: the same package path may describe a legacy project in one version and
// a runtime project in another.
func (r *Resolver) versionedPackageHasRuntime(ctx context.Context, target string) (bool, error) {
	importPath, query, ok := splitVersionedPackageTarget(target)
	if !ok {
		return false, nil
	}

	probeDir, err := os.MkdirTemp("", "xgo-runtime-version-probe-")
	if err != nil {
		return false, fmt.Errorf("create versioned package graph probe: %w", err)
	}
	defer os.RemoveAll(probeDir)
	probeMod := "module xgo.dev/runtimeprovider/versionprobe\n\ngo 1.25\n"
	if err := os.WriteFile(filepath.Join(probeDir, "go.mod"), []byte(probeMod), 0o600); err != nil {
		return false, fmt.Errorf("write versioned package graph probe: %w", err)
	}
	probeEnv := replaceEnv(os.Environ(), "GOWORK", "off")
	probeEnv = replaceEnv(probeEnv, "GOFLAGS", "-mod=mod")
	get := commandContext(ctx, r.policy.graph.GoCommand, "get", importPath+"@"+query)
	get.Dir = probeDir
	get.Env = probeEnv
	get.Stdout = io.Discard
	getErr := get.Run()
	if getErr != nil {
		// A failed go get remains the legacy path's diagnostic responsibility.
		// XGo-only packages may not be Go packages at all, so continue with the
		// structured module probe below without inferring anything from stderr.
		if cause := context.Cause(ctx); cause != nil {
			return false, cause
		}
	}

	probePolicy := GraphPolicy{
		GoCommand: r.policy.graph.GoCommand,
		GoWork:    "off",
		ModMode:   modModeMod,
		WorkDir:   probeDir,
	}
	var pkg goListPackage
	if getErr == nil {
		pkg, err = listPackageTarget(ctx, importPath, probeDir, probePolicy)
	}
	var probeResult versionedProbeResult
	if err == nil && pkg.ImportPath == importPath && pkg.Module != nil && pkg.Dir != "" {
		listed, normalizeErr := normalizeListedModule(*pkg.Module)
		err = normalizeErr
		if err == nil {
			projectDir, dirErr := canonicalExistingDir(pkg.Dir)
			err = dirErr
			if err == nil && pathWithin(listed.Effective().Dir, projectDir) {
				probeResult = versionedProbeResult{
					state:      versionedProbeMatch,
					module:     listed,
					projectDir: projectDir,
				}
			}
		}
	}
	if probeResult.state != versionedProbeMatch {
		if cause := context.Cause(ctx); cause != nil {
			return false, cause
		}
		probeResult, err = r.resolveVersionedModuleSource(ctx, probeDir, importPath, query, probeEnv)
		if err != nil {
			if cause := context.Cause(ctx); cause != nil {
				return false, cause
			}
			return false, fmt.Errorf("resolve versioned package %q module: %w", target, err)
		}
		if probeResult.state != versionedProbeMatch {
			return false, nil
		}
	}
	listed := probeResult.module
	projectDir := probeResult.projectDir
	if err := listed.Validate(); err != nil {
		return false, fmt.Errorf("validate versioned package %q module: %w", target, err)
	}
	probeMod = fmt.Sprintf("module xgo.dev/runtimeprovider/versionprobe\n\ngo 1.25\n\nrequire %s %s //xgo:class\n", listed.Selected.Path, listed.Selected.Version)
	if err := os.WriteFile(filepath.Join(probeDir, "go.mod"), []byte(probeMod), 0o600); err != nil {
		return false, fmt.Errorf("write versioned package graph probe: %w", err)
	}

	graph, err := loadEffectiveGraph(ctx, probeDir, probePolicy)
	if err != nil {
		if cause := context.Cause(ctx); cause != nil {
			return false, cause
		}
		return false, fmt.Errorf("load versioned package %q graph: %w", target, err)
	}
	requested, ok := graph.Modules[listed.Selected.Path]
	if !ok || !sameModuleSelection(requested, listed) {
		return false, fmt.Errorf("versioned package %q graph selected unexpected module %#v", target, requested)
	}
	if requested.Effective().Dir == "" || requested.Effective().GoMod == "" {
		// `go list -m all` may omit source fields for an otherwise selected
		// dependency. `go list -find` is authoritative for this package and has
		// already supplied a fully materialized module record.
		requested = listed
		graph.Modules[listed.Selected.Path] = requested
	}
	_, classPaths, err := readTargetModFile(requested.Effective().GoMod)
	if err != nil {
		return false, fmt.Errorf("read versioned package %q module metadata: %w", target, err)
	}
	for _, classPath := range classPaths {
		classModule, ok := graph.Modules[classPath]
		if !ok {
			return false, fmt.Errorf("versioned package %q class module %q is absent from its isolated graph", target, classPath)
		}
		if classModule.Effective().Dir == "" || classModule.Effective().GoMod == "" {
			classModule, err = downloadGraphModule(ctx, probeDir, probePolicy, classModule)
			if err != nil {
				if cause := context.Cause(ctx); cause != nil {
					return false, cause
				}
				return false, fmt.Errorf("materialize versioned package %q class module %q: %w", target, classPath, err)
			}
			graph.Modules[classPath] = classModule
		}
	}
	graph, err = retargetEffectiveGraph(graph, requested)
	if err != nil {
		return false, fmt.Errorf("retarget versioned package %q graph: %w", target, err)
	}
	module, _, err := loadResolvedClasses(graph)
	if err != nil {
		return false, fmt.Errorf("load versioned package %q runtime metadata: %w", target, err)
	}
	_, info, _, err := findProjectFile(projectDir, module)
	if err != nil {
		return false, fmt.Errorf("classify versioned package %q: %w", target, err)
	}
	return info != nil && info.Project != nil && info.Project.Runtime != nil, nil
}

type versionedProbeState uint8

const (
	versionedProbeMiss versionedProbeState = iota
	versionedProbeMatch
	versionedProbeNestedBoundary
)

type versionedProbeResult struct {
	state      versionedProbeState
	module     ResolvedModule
	projectDir string
}

func (r *Resolver) resolveVersionedModuleSource(ctx context.Context, probeDir, importPath, query string, env []string) (versionedProbeResult, error) {
	parts := strings.Split(importPath, "/")
	for length := len(parts); length > 0; length-- {
		candidate := strings.Join(parts[:length], "/")
		if err := module.CheckPath(candidate); err != nil {
			continue
		}
		result, err := r.downloadVersionedModule(ctx, probeDir, candidate, importPath, query, env)
		if err != nil {
			return versionedProbeResult{}, err
		}
		if result.state == versionedProbeNestedBoundary || result.state == versionedProbeMatch {
			return result, nil
		}
	}
	return versionedProbeResult{state: versionedProbeMiss}, nil
}

func (r *Resolver) downloadVersionedModule(ctx context.Context, probeDir, candidate, importPath, query string, env []string) (versionedProbeResult, error) {
	cmd := commandContext(ctx, r.policy.graph.GoCommand, "mod", "download", "-json", candidate+"@"+query)
	cmd.Dir = probeDir
	cmd.Env = env
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		if cause := context.Cause(ctx); cause != nil {
			return versionedProbeResult{}, cause
		}
		return versionedProbeResult{state: versionedProbeMiss}, nil
	}
	var downloaded goDownloadModule
	if err := json.Unmarshal(stdout.Bytes(), &downloaded); err != nil {
		return versionedProbeResult{}, fmt.Errorf("decode downloaded module %q: %w", candidate, err)
	}
	if downloaded.Error != "" || downloaded.Path != candidate || downloaded.Version == "" || downloaded.Dir == "" || downloaded.GoMod == "" {
		return versionedProbeResult{state: versionedProbeMiss}, nil
	}
	sourceDir, goMod, err := canonicalModuleSource(downloaded.Path, downloaded.Dir, downloaded.GoMod)
	if err != nil {
		return versionedProbeResult{state: versionedProbeMiss}, nil
	}
	state, projectDir, err := versionedModulePackageDir(sourceDir, downloaded.Path, importPath)
	if err != nil {
		return versionedProbeResult{}, err
	}
	if state != versionedProbeMatch {
		return versionedProbeResult{state: state}, nil
	}
	return versionedProbeResult{
		state:      versionedProbeMatch,
		projectDir: projectDir,
		module: ResolvedModule{
			Selected: ModuleRef{
				Path:    downloaded.Path,
				Version: downloaded.Version,
				Dir:     sourceDir,
				GoMod:   goMod,
			},
		},
	}, nil
}

func versionedModulePackageDir(moduleRoot, modulePath, importPath string) (versionedProbeState, string, error) {
	if !moduleContainsPackage(modulePath, importPath) {
		return versionedProbeMiss, "", nil
	}
	rel := strings.TrimPrefix(importPath, modulePath)
	rel = strings.TrimPrefix(rel, "/")
	projectDir, err := canonicalExistingDir(filepath.Join(moduleRoot, filepath.FromSlash(rel)))
	if err != nil {
		if os.IsNotExist(err) {
			return versionedProbeMiss, "", nil
		}
		return versionedProbeMiss, "", err
	}
	if !pathWithin(moduleRoot, projectDir) {
		return versionedProbeMiss, "", fmt.Errorf("package %q escapes module %q", importPath, modulePath)
	}
	for current := projectDir; current != moduleRoot; current = filepath.Dir(current) {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return versionedProbeNestedBoundary, "", nil
		} else if !os.IsNotExist(err) {
			return versionedProbeMiss, "", err
		}
	}
	return versionedProbeMatch, projectDir, nil
}

func splitVersionedPackageTarget(target string) (importPath, query string, ok bool) {
	index := strings.LastIndexByte(target, '@')
	if index <= 0 || index == len(target)-1 {
		return "", "", false
	}
	return target[:index], target[index+1:], true
}

func sameModuleSelection(a, b ResolvedModule) bool {
	if a.Main != b.Main || a.Selected.Path != b.Selected.Path || a.Selected.Version != b.Selected.Version {
		return false
	}
	if (a.Replace == nil) != (b.Replace == nil) {
		return false
	}
	if a.Replace == nil {
		return true
	}
	return a.Replace.Path == b.Replace.Path && a.Replace.Version == b.Replace.Version
}

// resolveOverlayLocalTarget disambiguates a local argument that does not exist
// on disk. xgoprojs parses both an overlay-only directory and an overlay-only
// file as DirProj, so classification must consult the same virtual filesystem
// as the Go command before deciding that the target belongs to the legacy path.
func (r *Resolver) resolveOverlayLocalTarget(ctx context.Context, candidate string, recursive bool) (TargetKind, string, string, *effectiveGraph, error) {
	graph, err := loadEffectiveGraph(ctx, r.cwd, r.policy.graph)
	if err != nil {
		if errors.Is(err, errNoGoModule) {
			return 0, "", "", nil, ErrNotHandled
		}
		return 0, "", "", nil, err
	}
	logical := overlayPath(r.cwd, candidate)
	projectDir, dirErr := graph.files.canonicalDir(logical)
	kind := TargetDirectory
	expectedFile := ""
	if dirErr != nil {
		if recursive {
			if os.IsNotExist(dirErr) {
				return 0, "", "", nil, ErrNotHandled
			}
			return 0, "", "", nil, fmt.Errorf("overlay local target %q: %w", candidate, dirErr)
		}
		visible, fileErr := graph.files.regularFileVisible(logical)
		if fileErr != nil {
			return 0, "", "", nil, fmt.Errorf("overlay local target %q: %w", candidate, fileErr)
		}
		if !visible {
			if !os.IsNotExist(dirErr) {
				return 0, "", "", nil, fmt.Errorf("overlay local target %q: %w", candidate, dirErr)
			}
			return 0, "", "", nil, ErrNotHandled
		}
		kind = TargetFile
		expectedFile = logical
		projectDir, err = graph.files.canonicalDir(filepath.Dir(logical))
		if err != nil {
			return 0, "", "", nil, fmt.Errorf("overlay local file target %q: %w", candidate, err)
		}
	}
	targetModule, err := graphModuleContainingDirectory(graph, projectDir)
	if err != nil {
		return 0, "", "", nil, err
	}
	graph, err = retargetEffectiveGraph(graph, targetModule)
	if err != nil {
		return 0, "", "", nil, err
	}
	return kind, projectDir, expectedFile, graph, nil
}

func graphModuleContainingDirectory(graph *effectiveGraph, dir string) (ResolvedModule, error) {
	var match ResolvedModule
	bestRoot := ""
	for _, module := range graph.Modules {
		root := module.Effective().Dir
		if root != "" && pathWithin(root, dir) && len(root) > len(bestRoot) {
			match = module
			bestRoot = root
		}
	}
	if bestRoot == "" {
		return ResolvedModule{}, fmt.Errorf("overlay target directory %q is outside the effective module graph", dir)
	}
	return match, nil
}

func classModuleMarked(classModules []ResolvedModule, modulePath string) bool {
	for _, module := range classModules {
		if module.Selected.Path == modulePath {
			return true
		}
	}
	return false
}

func (r *Resolver) preflightClassMetadata(ctx context.Context, dir string) (hasClass, vendor bool, err error) {
	_, _, hasClass, vendor, err = r.preflightClassMetadataDetails(ctx, dir)
	return
}

// preflightClassMetadataDetails reads only the target module metadata. It is
// intentionally separate from loadEffectiveGraph: go list -m all cannot run
// in vendor mode, and probing a legacy target must not turn that limitation
// into a runtime-provider error before a runtime project has been identified.
func (r *Resolver) preflightClassMetadataDetails(ctx context.Context, dir string) (loaded modload.Module, moduleGoMod string, hasClass, vendor bool, err error) {
	goMod, err := goEnvWithPolicy(ctx, r.policy.graph, dir, "GOMOD")
	if err != nil {
		return loaded, "", false, false, err
	}
	if goMod == "" || goMod == os.DevNull {
		return loaded, "", false, false, errNoGoModule
	}
	moduleGoMod, err = canonicalExistingFile(goMod)
	if err != nil {
		return loaded, "", false, false, err
	}
	effectiveMod := moduleGoMod
	if r.policy.graph.ModFile != "" {
		effectiveMod = r.policy.graph.ModFile
	}
	identity, classMods, err := readTargetModFile(effectiveMod)
	if err != nil {
		return loaded, moduleGoMod, false, false, err
	}
	loaded, err = loadRuntimeModule(identity.Path, filepath.Join(filepath.Dir(moduleGoMod), "gox.mod"))
	if err != nil {
		return loaded, moduleGoMod, false, false, err
	}
	hasClass = len(classMods) != 0 || loaded.HasProject()
	vendor, err = effectiveVendorMode(r.policy.graph, moduleGoMod, loaded.File)
	return loaded, moduleGoMod, hasClass, vendor, err
}

// probeVendorProject identifies a runtime project without invoking
// "go list -m all". Standard Go vendor snapshots do not preserve gox.mod or
// gop.mod reliably, so an external class marker is indeterminate and must fail
// closed instead of consulting a live replacement or returning ErrNotHandled.
func (r *Resolver) probeVendorProject(projectDir string, target modload.Module, recursive bool) (bool, error) {
	if classPath := externalClassModule(target); classPath != "" {
		return false, r.vendorClassMetadataError(classPath)
	}
	return matchVendorProjects(projectDir, target.Projects(), recursive)
}

// probeVendorPackage uses package-specific `go list`, which remains available
// in vendor mode, to distinguish another workspace main module from an
// unmarked dependency. Only main/workspace module metadata is authoritative;
// external class metadata is absent from standard vendor data and fails closed.
func (r *Resolver) probeVendorPackage(ctx context.Context, moduleGoMod string, target modload.Module, importPath string, recursive bool) (bool, error) {
	if classPath := externalClassModule(target); classPath != "" {
		return false, r.vendorClassMetadataError(classPath)
	}
	pkg, err := listPackageTarget(ctx, importPath, r.cwd, r.policy.graph)
	if err != nil {
		return false, err
	}
	if pkg.ImportPath != importPath {
		return false, fmt.Errorf("package target %q resolved as %q", importPath, pkg.ImportPath)
	}
	if pkg.Dir == "" || pkg.Module == nil {
		if moduleContainsPackage(target.Path(), importPath) {
			return r.probeVendorPackageInModule(ctx, moduleGoMod, target, importPath, recursive)
		}
		if r.policy.graph.GoWork != "off" {
			return r.probeVendorWorkspacePackage(ctx, importPath, recursive)
		}
		return false, nil
	}
	if !pkg.Module.Main {
		// An unmarked dependency cannot expand the provider trust boundary.
		return false, nil
	}
	module, err := normalizeListedModule(*pkg.Module)
	if err != nil {
		return false, fmt.Errorf("package target %q: %w", importPath, err)
	}
	root := module.Effective().Dir
	projectDir, err := canonicalExistingDir(pkg.Dir)
	if err != nil {
		return false, fmt.Errorf("package target %q: %w", importPath, err)
	}
	if !pathWithin(root, projectDir) {
		return false, fmt.Errorf("package target %q escapes module %q", importPath, module.Selected.Path)
	}
	loaded, err := loadRuntimeModule(module.Effective().GoMod, filepath.Join(root, "gox.mod"))
	if err != nil {
		return false, err
	}
	if classPath := externalClassModule(loaded); classPath != "" {
		return false, r.vendorClassMetadataError(classPath)
	}
	return matchVendorProjects(projectDir, loaded.Projects(), recursive)
}

func (r *Resolver) probeVendorPackageInModule(ctx context.Context, moduleGoMod string, target modload.Module, importPath string, recursive bool) (bool, error) {
	root := filepath.Dir(moduleGoMod)
	suffix := strings.TrimPrefix(importPath, target.Path())
	dir := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(suffix, "/")))
	projectDir, err := canonicalExistingDir(dir)
	if err != nil || !pathWithin(root, projectDir) {
		return false, nil
	}
	ownerGoMod, err := goEnvWithPolicy(ctx, r.policy.graph, projectDir, "GOMOD")
	if err != nil {
		return false, err
	}
	if ownerGoMod == "" || ownerGoMod == os.DevNull {
		return false, nil
	}
	same, err := sameFile(moduleGoMod, ownerGoMod)
	if err != nil {
		return false, err
	}
	if !same {
		return false, nil
	}
	return matchVendorProjects(projectDir, target.Projects(), recursive)
}

type workspaceVendorMember struct {
	modulePath string
	root       string
	goMod      string
}

// probeVendorWorkspacePackage handles XGo-only packages for which go list
// cannot report physical package fields. Only go.work use members are eligible:
// workspace replacements and other live dependency sources are deliberately
// excluded from this conservative vendor-mode classification.
func (r *Resolver) probeVendorWorkspacePackage(ctx context.Context, importPath string, recursive bool) (bool, error) {
	members, err := loadWorkspaceVendorMembers(r.policy.graph.GoWork)
	if err != nil {
		return false, err
	}
	var selected *workspaceVendorMember
	for i := range members {
		member := &members[i]
		if !moduleContainsPackage(member.modulePath, importPath) {
			continue
		}
		if selected == nil || len(member.modulePath) > len(selected.modulePath) {
			selected = member
		}
	}
	if selected == nil {
		return false, fmt.Errorf("%w: package target %q is not owned by a workspace member", vendorUnsupportedError(string(r.policy.graph.ModMode)), importPath)
	}

	loaded, err := loadRuntimeModule(selected.goMod, filepath.Join(selected.root, "gox.mod"))
	if err != nil {
		return false, fmt.Errorf("load workspace member %q metadata: %w", selected.modulePath, err)
	}
	if loaded.Path() != selected.modulePath {
		return false, fmt.Errorf("workspace member %q module path changed to %q during runtime discovery", selected.modulePath, loaded.Path())
	}
	if classPath := externalClassModule(loaded); classPath != "" {
		return false, r.vendorClassMetadataError(classPath)
	}

	suffix := strings.TrimPrefix(importPath, selected.modulePath)
	candidate := filepath.Join(selected.root, filepath.FromSlash(strings.TrimPrefix(suffix, "/")))
	projectDir, err := canonicalExistingDir(candidate)
	if err != nil {
		return false, fmt.Errorf("%w: package target %q has no classifiable workspace directory: %v", vendorUnsupportedError(string(r.policy.graph.ModMode)), importPath, err)
	}
	if !pathWithin(selected.root, projectDir) {
		return false, fmt.Errorf("package target %q escapes workspace module %q", importPath, selected.modulePath)
	}
	ownerGoMod, err := goEnvWithPolicy(ctx, r.policy.graph, projectDir, "GOMOD")
	if err != nil {
		return false, fmt.Errorf("resolve package target %q module ownership: %w", importPath, err)
	}
	if ownerGoMod == "" || ownerGoMod == os.DevNull {
		return false, fmt.Errorf("package target %q has no module ownership", importPath)
	}
	same, err := sameFile(selected.goMod, ownerGoMod)
	if err != nil {
		return false, fmt.Errorf("resolve package target %q module ownership: %w", importPath, err)
	}
	if !same {
		return false, fmt.Errorf("package target %q crosses a nested module boundary", importPath)
	}
	return matchVendorProjects(projectDir, loaded.Projects(), recursive)
}

func loadWorkspaceVendorMembers(goWork string) ([]workspaceVendorMember, error) {
	data, err := os.ReadFile(goWork)
	if err != nil {
		return nil, fmt.Errorf("read workspace file %q: %w", goWork, err)
	}
	work, err := gomodfile.ParseWork(goWork, data, nil)
	if err != nil {
		return nil, err
	}
	workRoot := filepath.Dir(goWork)
	members := make([]workspaceVendorMember, 0, len(work.Use))
	seenRoots := make(map[string]struct{}, len(work.Use))
	seenModules := make(map[string]string, len(work.Use))
	for _, use := range work.Use {
		if use == nil || use.Path == "" {
			return nil, fmt.Errorf("workspace %q contains an empty use path", goWork)
		}
		root := filepath.FromSlash(use.Path)
		if !filepath.IsAbs(root) {
			root = filepath.Join(workRoot, root)
		}
		root, err = canonicalExistingDir(root)
		if err != nil {
			return nil, fmt.Errorf("resolve workspace member %q: %w", use.Path, err)
		}
		if _, duplicate := seenRoots[root]; duplicate {
			return nil, fmt.Errorf("workspace %q contains duplicate member directory %q", goWork, root)
		}
		seenRoots[root] = struct{}{}

		goModPath := filepath.Join(root, "go.mod")
		info, err := os.Lstat(goModPath)
		if err != nil {
			return nil, fmt.Errorf("inspect workspace member go.mod %q: %w", goModPath, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("workspace member go.mod %q is not a regular non-symlink file", goModPath)
		}
		originalGoModPath := goModPath
		goModPath, err = canonicalExistingFile(originalGoModPath)
		if err != nil {
			return nil, fmt.Errorf("resolve workspace member go.mod %q: %w", originalGoModPath, err)
		}
		if filepath.Dir(goModPath) != root {
			return nil, fmt.Errorf("workspace member go.mod %q escapes member root %q", goModPath, root)
		}
		goModData, err := os.ReadFile(goModPath)
		if err != nil {
			return nil, fmt.Errorf("read workspace member go.mod %q: %w", goModPath, err)
		}
		parsed, err := gomodfile.Parse(goModPath, goModData, nil)
		if err != nil {
			return nil, err
		}
		if parsed.Module == nil || parsed.Module.Mod.Path == "" {
			return nil, fmt.Errorf("workspace member %q has no module path", root)
		}
		modulePath := parsed.Module.Mod.Path
		if previous, duplicate := seenModules[modulePath]; duplicate {
			return nil, fmt.Errorf("workspace module %q is declared by both %q and %q", modulePath, previous, root)
		}
		seenModules[modulePath] = root
		members = append(members, workspaceVendorMember{modulePath: modulePath, root: root, goMod: goModPath})
	}
	return members, nil
}

// loadRuntimeModule preserves modload's gox.mod-to-gop.mod fallback while
// distinguishing an absent optional metadata file from an unreadable one.
// Runtime ownership must not fall back to the legacy path on metadata I/O
// failures.
func loadRuntimeModule(goMod, goxMod string) (modload.Module, error) {
	return loadRuntimeModuleView(goMod, goxMod, nil)
}

func loadRuntimeModuleView(goMod, goxMod string, view *graphFileView) (modload.Module, error) {
	optionalPaths := map[string]struct{}{goxMod: {}}
	if strings.HasSuffix(goxMod, "gox.mod") {
		optionalPaths[strings.TrimSuffix(goxMod, "gox.mod")+"gop.mod"] = struct{}{}
	}
	var optionalErr error
	loaded, err := modload.LoadFromEx(goMod, goxMod, func(path string) ([]byte, error) {
		data, readErr := view.readFile(path)
		if _, optional := optionalPaths[path]; optional && readErr != nil && !os.IsNotExist(readErr) && optionalErr == nil {
			optionalErr = fmt.Errorf("read optional module metadata %q: %w", path, readErr)
		}
		return data, readErr
	})
	if optionalErr != nil {
		return modload.Module{}, optionalErr
	}
	return loaded, err
}

func externalClassModule(module modload.Module) string {
	for _, require := range module.Require {
		if require.Syntax != nil && modload.HasClassMarker(require.Syntax.Suffix) {
			return require.Mod.Path
		}
	}
	return ""
}

func (r *Resolver) vendorClassMetadataError(modulePath string) error {
	return fmt.Errorf("%w: class module %q metadata is not represented by standard Go vendor data", vendorUnsupportedError(string(r.policy.graph.ModMode)), modulePath)
}

func matchVendorProjects(dir string, projects []*modfile.Project, recursive bool) (bool, error) {
	if recursive {
		return patternContainsRuntimeProjects(dir, projects)
	}
	return hasRuntimeProject(dir, projects)
}

func hasRuntimeProject(dir string, projects []*modfile.Project) (bool, error) {
	if len(projects) == 0 {
		return false, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return false, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		ext := modfile.ClassExt(entry.Name())
		for _, project := range projects {
			if project.Runtime != nil && project.IsProj(ext, entry.Name()) {
				return true, nil
			}
		}
	}
	return false, nil
}

func graphHasClassMetadata(graph *effectiveGraph) (bool, error) {
	loaded, err := loadRuntimeModuleView(graph.TargetModFile.Path, filepath.Join(graph.Target.Effective().Dir, "gox.mod"), graph.files)
	if err != nil {
		return false, err
	}
	return len(graph.ClassModules) != 0 || loaded.HasProject(), nil
}

// overlayRuntimeProjectMatch classifies a target using the effective graph's
// overlay-aware metadata. It is intentionally a classification-only path:
// v1 has no snapshot contract for overlays, so a positive match is converted
// to an explicit unsupported error before any provider Runtime is built.
func overlayRuntimeProjectMatch(projectDir string, graph *effectiveGraph, recursive bool) (bool, error) {
	if graph == nil || graph.files == nil || !graph.files.hasOverlay() {
		return false, fmt.Errorf("overlay classification requires an effective graph file view")
	}
	projects, err := overlayRuntimeProjects(graph)
	if err != nil {
		return false, err
	}
	if len(projects) == 0 {
		return false, nil
	}
	if recursive {
		return overlayWalkRuntimeProjects(projectDir, projects, graph.files)
	}
	names, err := graph.files.regularFileNames(projectDir)
	if err != nil {
		return false, err
	}
	for _, name := range names {
		ext := modfile.ClassExt(name)
		for _, project := range projects {
			if project.Runtime != nil && project.IsProj(ext, name) {
				return true, nil
			}
		}
	}
	return false, nil
}

func overlayRuntimeProjects(graph *effectiveGraph) ([]*modfile.Project, error) {
	target := graph.Target.Effective()
	loaded, err := loadRuntimeModuleView(graph.TargetModFile.Path, filepath.Join(target.Dir, "gox.mod"), graph.files)
	if err != nil {
		return nil, fmt.Errorf("load overlaid target metadata: %w", err)
	}
	projects := append([]*modfile.Project(nil), loaded.Projects()...)
	for _, class := range graph.ClassModules {
		effective := class.Effective()
		classModule, loadErr := loadRuntimeModuleView(effective.GoMod, filepath.Join(effective.Dir, "gox.mod"), graph.files)
		if loadErr != nil {
			return nil, fmt.Errorf("load overlaid class module %q metadata: %w", class.Selected.Path, loadErr)
		}
		projects = append(projects, classModule.Projects()...)
	}
	return projects, nil
}

func overlayWalkRuntimeProjects(root string, projects []*modfile.Project, view *graphFileView) (bool, error) {
	root = overlayPath(view.workDir, root)
	visited := make(map[string]struct{})
	var walk func(string) error
	walk = func(current string) error {
		if _, ok := visited[current]; ok {
			return nil
		}
		visited[current] = struct{}{}
		names, err := view.regularFileNames(current)
		if err != nil {
			return err
		}
		for _, name := range names {
			ext := modfile.ClassExt(name)
			for _, project := range projects {
				if project.Runtime != nil && project.IsProj(ext, name) {
					return errRuntimeProjectInPattern
				}
			}
		}
		dirs, err := view.directoryNames(current)
		if err != nil {
			return err
		}
		for _, name := range dirs {
			if name == "vendor" || name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				continue
			}
			child := filepath.Join(current, name)
			if isGoMod, err := view.regularFileVisible(filepath.Join(child, "go.mod")); err != nil {
				return err
			} else if isGoMod {
				continue
			}
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	err := walk(root)
	if errors.Is(err, errRuntimeProjectInPattern) {
		return true, nil
	}
	return false, err
}

func unsupportedOverlayError(path string) error {
	return fmt.Errorf("runtime provider v1 does not support flag -overlay=%s", path)
}

func goEnvWithPolicy(ctx context.Context, policy GraphPolicy, dir, key string) (string, error) {
	args := []string{"env", key}
	cmd := commandContext(ctx, policy.GoCommand, args...)
	cmd.Dir = dir
	// Graph flags do not affect go env values and are deliberately never
	// reconstructed into GOFLAGS; all graph operations pass them via argv.
	cmd.Env = graphEnvironment(os.Environ(), policy.GoWork)
	out, err := cmd.Output()
	if err != nil {
		return "", commandError("go env "+key, err, string(cmdStderr(cmd)))
	}
	return strings.TrimSpace(string(out)), nil
}

func loadResolvedClasses(graph *effectiveGraph) (*xgomod.Module, bool, error) {
	target := graph.Target.Effective()
	loaded, err := loadRuntimeModule(graph.TargetModFile.Path, filepath.Join(target.Dir, "gox.mod"))
	if err != nil {
		return nil, false, err
	}
	module := xgomod.New(loaded)
	hasRuntime := false
	if err := module.ImportClassesResolved(toXGoGraph(graph), func(info *xgomod.ProjectInfo) {
		if info != nil && info.Project != nil && info.Project.Runtime != nil {
			hasRuntime = true
		}
	}); err != nil {
		return nil, false, err
	}
	return module, hasRuntime, nil
}

func toXGoGraph(graph *effectiveGraph) xgomod.ResolvedClassGraph {
	// xgomod only consumes the target and explicitly marked class modules.
	// Keep the complete build list in effectiveGraph for package resolution,
	// while avoiding irrelevant legacy modules whose standard go list GoMod
	// identity may live in cache/download rather than the extracted source.
	return xgomod.ResolvedClassGraph{
		Target: graph.Target, ClassModules: graph.ClassModules, TargetModFile: graph.TargetModFile,
	}
}

func findProjectFile(dir string, module *xgomod.Module) (string, *xgomod.ProjectInfo, int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, 0, err
	}
	var (
		projectFile string
		projectInfo *xgomod.ProjectInfo
		count       int
	)
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return "", nil, 0, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		ext := modfile.ClassExt(entry.Name())
		classInfo, ok := module.LookupClassInfo(ext)
		if !ok || !classInfo.Project.IsProj(ext, entry.Name()) {
			continue
		}
		count++
		if classInfo.Project.Runtime != nil {
			projectFile = filepath.Join(dir, entry.Name())
			projectInfo = classInfo
		}
	}
	return projectFile, projectInfo, count, nil
}

var errRuntimeProjectInPattern = errors.New("runtime project in pattern")

func patternContainsRuntimeProject(root string, module *xgomod.Module) (bool, error) {
	return walkRuntimePattern(root, func(dir string) (bool, error) {
		_, info, _, err := findProjectFile(dir, module)
		return info != nil && info.Project != nil && info.Project.Runtime != nil, err
	})
}

func patternContainsRuntimeProjects(root string, projects []*modfile.Project) (bool, error) {
	return walkRuntimePattern(root, func(dir string) (bool, error) {
		return hasRuntimeProject(dir, projects)
	})
}

func walkRuntimePattern(root string, matches func(string) (bool, error)) (bool, error) {
	err := filepath.WalkDir(root, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		if current != root {
			name := entry.Name()
			if entry.Type()&os.ModeSymlink != 0 || name == "vendor" || name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				return filepath.SkipDir
			}
			goMod := filepath.Join(current, "go.mod")
			if info, err := os.Lstat(goMod); err == nil {
				if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
					return fmt.Errorf("nested module marker %q is not a regular non-symlink file", goMod)
				}
				return filepath.SkipDir
			} else if !os.IsNotExist(err) {
				return err
			}
		}
		matched, err := matches(current)
		if err != nil {
			return err
		}
		if matched {
			return errRuntimeProjectInPattern
		}
		return nil
	})
	if errors.Is(err, errRuntimeProjectInPattern) {
		return true, nil
	}
	return false, err
}

func validatePack(projectDir string, pack *modfile.Pack) (string, string, error) {
	if pack == nil {
		return "", "", nil
	}
	dir := filepath.Clean(filepath.FromSlash(pack.Directory))
	if pack.Directory == "" || filepath.IsAbs(dir) || dir == ".." || strings.HasPrefix(dir, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("runtime pack directory %q is invalid", pack.Directory)
	}
	if pack.IndexFile == "" || filepath.Base(pack.IndexFile) != pack.IndexFile || strings.ContainsAny(pack.IndexFile, `/\`) {
		return "", "", fmt.Errorf("runtime pack index %q is invalid", pack.IndexFile)
	}
	root := filepath.Join(projectDir, dir)
	canonical, err := canonicalExistingDir(root)
	if err != nil {
		return "", "", fmt.Errorf("runtime pack directory: %w", err)
	}
	if !pathWithin(projectDir, canonical) {
		return "", "", fmt.Errorf("runtime pack directory escapes the project")
	}
	return filepath.ToSlash(dir), pack.IndexFile, nil
}

func declaringMetadata(origin ResolvedModule, snapshot xgomod.FileIdentity) (fileIdentity, error) {
	dir := origin.Effective().Dir
	base := filepath.Base(snapshot.Path)
	if snapshot.Path == "" || snapshot.SHA256 == "" || filepath.Dir(snapshot.Path) != dir || (base != "gox.mod" && base != "gop.mod") {
		return fileIdentity{}, fmt.Errorf("runtime origin %q has an invalid declaring metadata snapshot", origin.Selected.Path)
	}
	if len(snapshot.SHA256) != sha256.Size*2 {
		return fileIdentity{}, fmt.Errorf("runtime origin %q has an invalid declaring metadata digest", origin.Selected.Path)
	}
	if _, err := hex.DecodeString(snapshot.SHA256); err != nil || snapshot.SHA256 != strings.ToLower(snapshot.SHA256) {
		return fileIdentity{}, fmt.Errorf("runtime origin %q has an invalid declaring metadata digest", origin.Selected.Path)
	}
	before, err := os.Lstat(snapshot.Path)
	if err != nil {
		return fileIdentity{}, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return fileIdentity{}, fmt.Errorf("declaring metadata %q is not a regular non-symlink file", snapshot.Path)
	}
	file, err := os.Open(snapshot.Path)
	if err != nil {
		return fileIdentity{}, err
	}
	data, readErr := io.ReadAll(file)
	opened, statErr := file.Stat()
	closeErr := file.Close()
	if readErr != nil {
		return fileIdentity{}, readErr
	}
	if statErr != nil {
		return fileIdentity{}, statErr
	}
	if closeErr != nil {
		return fileIdentity{}, closeErr
	}
	after, err := os.Lstat(snapshot.Path)
	if err != nil || !os.SameFile(before, opened) || !os.SameFile(opened, after) || !after.Mode().IsRegular() {
		return fileIdentity{}, fmt.Errorf("declaring metadata %q changed after discovery", snapshot.Path)
	}
	if got := sha256Bytes(data); got != snapshot.SHA256 {
		return fileIdentity{}, fmt.Errorf("declaring metadata %q changed after discovery", snapshot.Path)
	}
	return fileIdentity(snapshot), nil
}

func sameFile(a, b string) (bool, error) {
	aInfo, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	bInfo, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return os.SameFile(aInfo, bInfo), nil
}

func effectiveVendorMode(policy GraphPolicy, moduleGoMod string, parsed *gomodfile.File) (bool, error) {
	if policy.ModMode != "" {
		return policy.ModMode == modModeVendor, nil
	}
	workspace := policy.GoWork != "" && policy.GoWork != "off"
	var (
		goVersion string
		vendorDir string
	)
	if workspace {
		data, err := os.ReadFile(policy.GoWork)
		if err != nil {
			return false, err
		}
		work, err := gomodfile.ParseWork(policy.GoWork, data, nil)
		if err != nil {
			return false, err
		}
		if work.Go != nil {
			goVersion = work.Go.Version
		}
		vendorDir = filepath.Join(filepath.Dir(policy.GoWork), "vendor")
	} else {
		if parsed.Go != nil {
			goVersion = parsed.Go.Version
		}
		vendorDir = filepath.Join(filepath.Dir(moduleGoMod), "vendor")
	}
	if goVersion == "" || !versionAtLeast(goVersion, 1, 14) {
		return false, nil
	}
	info, err := os.Stat(vendorDir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, nil
	}
	vendoredWorkspace, err := vendorManifestIsForWorkspace(vendorDir)
	if err != nil {
		return false, err
	}
	return vendoredWorkspace == workspace, nil
}

// vendorManifestIsForWorkspace mirrors cmd/go's modulesTextIsForWorkspace.
// A missing modules.txt retains the historical module-vendor behavior, but it
// cannot identify a workspace vendor directory.
func vendorManifestIsForWorkspace(vendorDir string) (bool, error) {
	file, err := os.Open(filepath.Join(vendorDir, "modules.txt"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer file.Close()
	var buf [512]byte
	n, err := file.Read(buf[:])
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	line, _, _ := strings.Cut(string(buf[:n]), "\n")
	annotations, ok := strings.CutPrefix(line, "## ")
	if !ok {
		return false, nil
	}
	for entry := range strings.SplitSeq(annotations, ";") {
		if strings.TrimSpace(entry) == "workspace" {
			return true, nil
		}
	}
	return false, nil
}

func versionAtLeast(version string, major, minor int) bool {
	var gotMajor, gotMinor int
	if _, err := fmt.Sscanf(version, "%d.%d", &gotMajor, &gotMinor); err != nil {
		return false
	}
	return gotMajor > major || gotMajor == major && gotMinor >= minor
}

func vendorUnsupportedError(mode string) error {
	if mode == "" {
		mode = "automatic vendor mode"
	} else {
		mode = "-mod=" + mode
	}
	return fmt.Errorf("%w (%s); select -mod=readonly or -mod=mod explicitly", ErrRuntimeVendorUnsupported, mode)
}

func defaultExecutableName(kind TargetKind, projectDir, importPath string) string {
	if kind != TargetPackage || importPath == "" {
		return filepath.Base(projectDir)
	}
	parts := strings.Split(strings.TrimSuffix(importPath, "/"), "/")
	name := parts[len(parts)-1]
	if majorVersionRE.MatchString(name) && len(parts) > 1 {
		name = parts[len(parts)-2]
	}
	return name
}

var majorVersionRE = regexp.MustCompile(`^v[2-9][0-9]*$`)

func hasRecursivePattern(target string) bool {
	clean := filepath.ToSlash(filepath.Clean(target))
	return clean == "..." || strings.HasSuffix(clean, "/...")
}

func trimRecursivePattern(target string) string {
	clean := filepath.ToSlash(filepath.Clean(target))
	clean = strings.TrimSuffix(clean, "...")
	clean = strings.TrimSuffix(clean, "/")
	if clean == "" {
		return "."
	}
	return filepath.FromSlash(clean)
}

func sha256Bytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
