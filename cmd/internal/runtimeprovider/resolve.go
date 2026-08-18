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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/goplus/mod/modfile"
	"github.com/goplus/mod/modload"
	"github.com/goplus/mod/xgomod"
	"github.com/goplus/xgo/env"
	"github.com/goplus/xgo/x/xgoprojs"
	gomodfile "golang.org/x/mod/modfile"
)

const runtimeGuardEnv = "XGO_RUNTIME_GUARD"

func runtimeGuard(projectDir, providerPackage string) string {
	return sha256Bytes([]byte(projectDir + "\x00" + providerPackage))
}

// Resolver owns one invocation's graph and host-build policies.
type Resolver struct {
	cwd        string
	policy     parsedFlags
	policyErr  error
	xgoVersion string
}

// NewResolver snapshots ambient GOFLAGS/GOWORK once for an invocation. A
// policy setup error is retained for BuildPolicy instead of being returned
// immediately: discovery must be able to return ErrNotHandled for an
// ordinary legacy target before runtime-only policy errors are surfaced.
func NewResolver(ctx context.Context, cwd string, flags []string) (*Resolver, error) {
	cwd, err := canonicalExistingDir(cwd)
	if err != nil {
		return nil, err
	}
	policy, err := preparePolicies(ctx, cwd, flags)
	if err == nil {
		return &Resolver{cwd: cwd, policy: policy, xgoVersion: env.Version()}, nil
	}
	// Use a conservative, no-ambient-policy fallback for discovery. The
	// retained error is checked only when execution asks for BuildPolicy.
	fallback := parsedFlags{graph: GraphPolicy{GoCommand: "go", GoWork: "off"}}
	if command, commandErr := hostGoCommand(); commandErr == nil {
		fallback.graph.GoCommand = command
	}
	if parsed, parseErr := parseRuntimeFlags(cwd, fallback.graph.GoCommand, "off", "", flags); parseErr == nil {
		fallback = sanitizeGraphFlags(parsed)
		if fallback.graph.GoWork == "" {
			fallback.graph.GoWork = "off"
		}
	}
	return &Resolver{cwd: cwd, policy: fallback, policyErr: err, xgoVersion: env.Version()}, nil
}

// BuildPolicy returns the validated runtime build policy. Callers must invoke
// it only after Resolve matched a runtime project.
func (r *Resolver) BuildPolicy() (BuildPolicy, error) {
	if r.policyErr != nil {
		return BuildPolicy{}, r.policyErr
	}
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
		graph            *effectiveGraph
		graphVendor      bool
		graphWorkDir     string
	)
	switch target := target.(type) {
	case *xgoprojs.DirProj:
		kind, original = TargetDirectory, target.Dir
		candidate := target.Dir
		if hasRecursivePattern(candidate) {
			unsupportedForm = "directory pattern containing ..."
			candidate = trimRecursivePattern(candidate)
		}
		var err error
		projectDir, err = canonicalExistingDir(candidate)
		if err != nil {
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
			unsupportedForm = "package target containing @version"
			lookupPath = strings.SplitN(lookupPath, "@", 2)[0]
		}
		if hasRecursivePattern(lookupPath) {
			unsupportedForm = "package pattern containing ..."
			lookupPath = trimRecursivePattern(lookupPath)
		}
		preflightModule, preflightGoMod, callerHasClass, vendor, err := r.preflightClassMetadataDetails(ctx, r.cwd)
		if err != nil && !errors.Is(err, errNoGoModule) {
			return nil, err
		}
		callerGraph, err := loadEffectiveGraph(ctx, r.cwd, r.policy.graph)
		if err != nil {
			if callerHasClass && vendor {
				hasRuntime, probeErr := r.probeVendorPackage(ctx, preflightGoMod, preflightModule, lookupPath)
				if probeErr != nil {
					return nil, probeErr
				}
				if hasRuntime {
					return nil, vendorUnsupportedError(r.policy.graph.ModMode)
				}
				return nil, ErrNotHandled
			}
			if errors.Is(err, errNoGoModule) && !callerHasClass {
				return nil, ErrNotHandled
			}
			return nil, err
		}
		var targetModule ResolvedModule
		projectDir, targetModule, err = resolvePackageDirectory(callerGraph, lookupPath)
		if err != nil {
			if callerHasClass {
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
		graphVendor = vendor
		graphWorkDir = r.cwd
	default:
		return nil, ErrNotHandled
	}
	graphPolicy := r.policy.graph
	graphPolicy.WorkDir = graphWorkDir

	if graph == nil {
		preflightModule, preflightGoMod, hasClass, vendor, err := r.preflightClassMetadataDetails(ctx, projectDir)
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
			hasRuntime, probeErr := r.probeVendorProject(ctx, projectDir, preflightGoMod, preflightModule)
			if probeErr != nil {
				return nil, probeErr
			}
			if !hasRuntime {
				return nil, ErrNotHandled
			}
			return nil, vendorUnsupportedError(r.policy.graph.ModMode)
		}
		graph, err = loadEffectiveGraph(ctx, projectDir, r.policy.graph)
		if err != nil {
			return nil, err
		}
	} else {
		hasClass, err := graphHasClassMetadata(graph)
		if err != nil {
			return nil, err
		}
		if !hasClass {
			return nil, ErrNotHandled
		}
	}
	module, hasRuntimeMetadata, err := loadResolvedClasses(graph)
	if err != nil {
		return nil, err
	}
	if unsupportedForm != "" && hasRuntimeMetadata {
		return nil, fmt.Errorf("runtime provider v1 does not support %s", unsupportedForm)
	}
	projectFile, info, candidates, err := findProjectFile(projectDir, module)
	if err != nil {
		return nil, err
	}
	if info == nil || info.Project.Runtime == nil {
		return nil, ErrNotHandled
	}
	if graphVendor {
		return nil, vendorUnsupportedError(r.policy.graph.ModMode)
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
	guard := runtimeGuard(projectDir, project.Runtime.Package)
	if os.Getenv(runtimeGuardEnv) == guard {
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
	if alternate := graphFlagValue(r.policy.graph.Flags, "modfile"); alternate != "" {
		effectiveMod = alternate
	}
	identity, classMods, err := readTargetModFile(effectiveMod)
	if err != nil {
		return loaded, moduleGoMod, false, false, err
	}
	loaded, err = modload.LoadFrom(identity.Path, filepath.Join(filepath.Dir(moduleGoMod), "gox.mod"))
	if err != nil {
		return loaded, moduleGoMod, false, false, err
	}
	hasClass = len(classMods) != 0 || loaded.HasProject()
	vendor, err = effectiveVendorMode(r.policy.graph, moduleGoMod, loaded.File)
	return loaded, moduleGoMod, hasClass, vendor, err
}

// vendorModuleCandidate is the small amount of module information needed by
// the vendor probe. It deliberately does not ask the Go command to load a
// complete module graph: that operation is not supported by vendor mode.
type vendorModuleCandidate struct {
	path string
	root string
	mod  modload.Module
}

// probeVendorProject identifies a runtime project without invoking
// "go list -m all". A class module that has no runtime project is therefore
// still returned as ErrNotHandled and follows the legacy path.
func (r *Resolver) probeVendorProject(ctx context.Context, projectDir, moduleGoMod string, target modload.Module) (bool, error) {
	if hasRuntimeProject(projectDir, target.Projects()) {
		return true, nil
	}
	for _, candidate := range r.vendorModuleCandidates(moduleGoMod, target) {
		if candidate.path == target.Path() {
			continue
		}
		if candidate.mod.HasModfile() && hasRuntimeProject(projectDir, candidate.mod.Projects()) {
			return true, nil
		}
	}
	return false, nil
}

// probeVendorPackage applies the same delayed runtime check to package
// targets. It first maps the package to a vendor/replacement module, then
// checks only the files in that package directory. Thus an ordinary Go
// subpackage inside a runtime module remains a legacy target.
func (r *Resolver) probeVendorPackage(ctx context.Context, moduleGoMod string, target modload.Module, importPath string) (bool, error) {
	for _, candidate := range r.vendorModuleCandidates(moduleGoMod, target) {
		if !moduleContainsPackage(candidate.path, importPath) {
			continue
		}
		suffix := strings.TrimPrefix(importPath, candidate.path)
		dir := filepath.Join(candidate.root, filepath.FromSlash(strings.TrimPrefix(suffix, "/")))
		canonical, err := canonicalExistingDir(dir)
		if err != nil || !pathWithin(candidate.root, canonical) {
			continue
		}
		if candidate.mod.HasModfile() && hasRuntimeProject(canonical, candidate.mod.Projects()) {
			return true, nil
		}
		// The longest matching module owns this import path. If it has no
		// runtime project, shorter module prefixes cannot change the result.
		return false, nil
	}
	return false, nil
}

func hasRuntimeProject(dir string, projects []*modfile.Project) bool {
	if len(projects) == 0 {
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		ext := modfile.ClassExt(entry.Name())
		for _, project := range projects {
			if project.Runtime != nil && project.IsProj(ext, entry.Name()) {
				return true
			}
		}
	}
	return false
}

func (r *Resolver) vendorModuleCandidates(moduleGoMod string, target modload.Module) []vendorModuleCandidate {
	moduleRoot := filepath.Dir(moduleGoMod)
	seen := make(map[string]bool)
	var candidates []vendorModuleCandidate
	add := func(path, root string, loaded *modload.Module) {
		if path == "" || seen[path] {
			return
		}
		root, err := canonicalExistingDir(root)
		if err != nil {
			return
		}
		var mod modload.Module
		if loaded != nil {
			mod = *loaded
		} else {
			goMod := filepath.Join(root, "go.mod")
			if _, err := os.Stat(goMod); err != nil {
				return
			}
			loadedMod, err := modload.LoadFrom(goMod, filepath.Join(root, "gox.mod"))
			if err != nil {
				return
			}
			mod = loadedMod
		}
		seen[path] = true
		candidates = append(candidates, vendorModuleCandidate{path: path, root: root, mod: mod})
	}
	add(target.Path(), moduleRoot, &target)

	local := make(map[string]string)
	for _, replacement := range target.Replace {
		if replacement.Old.Path == "" || replacement.New.Version != "" {
			continue
		}
		root := replacement.New.Path
		if !filepath.IsAbs(root) {
			root = filepath.Join(moduleRoot, root)
		}
		local[replacement.Old.Path] = root
	}
	vendorRoots := []string{filepath.Join(moduleRoot, "vendor")}
	if r.policy.graph.GoWork != "" && r.policy.graph.GoWork != "off" {
		vendorRoots = append(vendorRoots, filepath.Join(filepath.Dir(r.policy.graph.GoWork), "vendor"))
	}
	for _, require := range target.Require {
		if require.Syntax == nil || !modload.HasClassMarker(require.Syntax.Suffix) {
			continue
		}
		path := require.Mod.Path
		if path == "" {
			continue
		}
		if root, ok := local[path]; ok {
			add(path, root, nil)
			continue
		}
		for _, vendorRoot := range vendorRoots {
			add(path, filepath.Join(vendorRoot, filepath.FromSlash(path)), nil)
			if seen[path] {
				break
			}
		}
	}
	// Resolve the longest module prefix first, matching Go package lookup.
	sort.SliceStable(candidates, func(i, j int) bool {
		return len(candidates[i].path) > len(candidates[j].path)
	})
	return candidates
}

func graphHasClassMetadata(graph *effectiveGraph) (bool, error) {
	loaded, err := modload.LoadFrom(graph.TargetModFile.Path, filepath.Join(graph.Target.Effective().Dir, "gox.mod"))
	if err != nil {
		return false, err
	}
	return len(graph.ClassModules) != 0 || loaded.HasProject(), nil
}

func goEnvWithPolicy(ctx context.Context, policy GraphPolicy, dir, key string) (string, error) {
	args := []string{"env", key}
	cmd := commandContext(ctx, policy.GoCommand, args...)
	cmd.Dir = dir
	// Graph flags do not affect go env values and are deliberately never
	// reconstructed into GOFLAGS; all graph operations pass them via argv.
	cmd.Env = graphEnvironment(os.Environ(), policy.GoWork, nil)
	out, err := cmd.Output()
	if err != nil {
		return "", commandError("go env "+key, err, string(cmdStderr(cmd)))
	}
	return strings.TrimSpace(string(out)), nil
}

func loadResolvedClasses(graph *effectiveGraph) (*xgomod.Module, bool, error) {
	target := graph.Target.Effective()
	loaded, err := modload.LoadFrom(graph.TargetModFile.Path, filepath.Join(target.Dir, "gox.mod"))
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
		return policy.ModMode == "vendor", nil
	}
	if policy.GoWork != "off" {
		data, err := os.ReadFile(policy.GoWork)
		if err != nil {
			return false, err
		}
		work, err := gomodfile.ParseWork(policy.GoWork, data, nil)
		if err != nil {
			return false, err
		}
		if work.Go != nil && versionAtLeast(work.Go.Version, 1, 22) {
			_, err := os.Stat(filepath.Join(filepath.Dir(policy.GoWork), "vendor", "modules.txt"))
			return err == nil, nil
		}
		return false, nil
	}
	if parsed.Go == nil || !versionAtLeast(parsed.Go.Version, 1, 14) {
		return false, nil
	}
	_, err := os.Stat(filepath.Join(filepath.Dir(moduleGoMod), "vendor", "modules.txt"))
	return err == nil, nil
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
