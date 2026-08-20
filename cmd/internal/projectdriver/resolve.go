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

package projectdriver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goplus/mod/modload"
	"github.com/goplus/xgo/env"
	"github.com/goplus/xgo/x/xgoprojs"
)

// Resolver owns one invocation's graph and host-build policies.
type Resolver struct {
	cwd        string
	policy     parsedFlags
	xgoVersion string
}

// NewResolver snapshots ambient GOFLAGS/GOWORK once for an invocation. Policy
// setup must not fall back to a different module graph: doing so could classify
// a workspace driver-backed target as legacy and incorrectly dispatch it to GenGo.
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

// BuildPolicy returns the validated driver build policy. Callers must invoke
// it only after Resolve matched a driver-backed project.
func (r *Resolver) BuildPolicy() (BuildPolicy, error) {
	if err := r.policy.validateDriver(); err != nil {
		return BuildPolicy{}, err
	}
	return r.policy.build, nil
}

// Resolve resolves a parsed XGo target without parsing or generating source.
func (r *Resolver) Resolve(ctx context.Context, target xgoprojs.Proj) (*Driver, error) {
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
		err              error
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
			hasDriver, err := r.versionedPackageHasDriver(ctx, lookupPath)
			if err != nil {
				return nil, err
			}
			if !hasDriver {
				return nil, ErrNotHandled
			}
			return nil, fmt.Errorf("driver v1 does not support package target containing @version")
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
				// cannot select a driver. Avoid asking `go list -m all`,
				// which is unsupported in vendor mode, and preserve its legacy path.
				if !callerHasClass && r.policy.graph.GoWork == "off" {
					return nil, ErrNotHandled
				}
				hasDriver, probeErr := r.probeVendorPackage(ctx, preflightGoMod, preflightModule, lookupPath, recursivePattern)
				if probeErr != nil {
					return nil, probeErr
				}
				if hasDriver {
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
		// A package target may use driver metadata from its own main/workspace
		// module. A non-main dependency is eligible only when the caller's exact
		// target modfile snapshot marked that logical module as an XGo class
		// dependency. Do not let retargeting discover an unmarked dependency's
		// ambient gox.mod and expand the driver execution trust boundary.
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

	graph, err = r.resolveTargetGraph(ctx, projectDir, graph, graphPolicy, recursivePattern)
	if err != nil {
		return nil, err
	}
	module, hasClass, err := loadResolvedClasses(graph)
	if err != nil {
		return nil, err
	}
	if !hasClass {
		return nil, ErrNotHandled
	}
	if recursivePattern {
		hasDriver, err := patternContainsDriverProject(projectDir, module)
		if err != nil {
			return nil, err
		}
		if !hasDriver {
			return nil, ErrNotHandled
		}
		return nil, fmt.Errorf("driver v1 does not support %s", unsupportedForm)
	}
	projectFile, info, candidates, err := findProjectFile(projectDir, module)
	if err != nil {
		return nil, err
	}
	if info == nil || info.Project.Driver == nil {
		return nil, ErrNotHandled
	}
	if unsupportedForm != "" {
		return nil, fmt.Errorf("driver v1 does not support %s", unsupportedForm)
	}
	if candidates != 1 {
		return nil, fmt.Errorf("driver-backed project directory %q contains %d project files; exactly one is required", projectDir, candidates)
	}
	if multiFile {
		return nil, fmt.Errorf("driver v1 does not support multiple source-file targets")
	}
	if expectedFile != "" {
		same, err := sameFile(expectedFile, projectFile)
		if err != nil || !same {
			return nil, fmt.Errorf("driver file target %q is not the unique project file %q", original, projectFile)
		}
	}
	if info.Origin == nil {
		return nil, fmt.Errorf("driver-backed project %q has no module provenance", projectFile)
	}
	if err := checkRequiredXGo(info.RequiredXGo, r.xgoVersion); err != nil {
		return nil, err
	}
	project := info.Project
	if project.Driver.Protocol != protocolV1 {
		return nil, fmt.Errorf("unsupported driver protocol %q", project.Driver.Protocol)
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
	if os.Getenv("XGO_DRIVER") == "off" {
		return nil, ErrDriverDisabled
	}
	if os.Getenv(driverGuardEnv) != "" {
		return nil, ErrDriverRecursive
	}
	defaultName := defaultExecutableName(kind, projectDir, targetImportPath)
	return &Driver{
		TargetKind:       kind,
		OriginalTarget:   original,
		TargetImportPath: targetImportPath,
		DefaultExecName:  defaultName,
		ProjectDir:       projectDir,
		ProjectFile:      projectFile,
		ModuleRoot:       graph.Target.Effective().Dir,
		DriverPackage:    project.Driver.Package,
		Origin:           origin,
		RequiredXGo:      info.RequiredXGo,
		Protocol:         project.Driver.Protocol,
		ProjectExt:       project.Ext,
		ProjectFullExt:   project.FullExt,
		PackDir:          packDir,
		PackIndex:        packIndex,
		GoxMod:           goxmod.Path,
		GoxModSHA256:     goxmod.SHA256,
		Graph:            graphPolicy,
	}, nil
}

func (r *Resolver) resolveTargetGraph(ctx context.Context, projectDir string, graph *effectiveGraph, policy GraphPolicy, recursive bool) (*effectiveGraph, error) {
	if graph != nil {
		if graph.files == nil || !graph.files.hasOverlay() {
			return graph, nil
		}
		hasDriver, err := overlayDriverProjectMatch(projectDir, graph, recursive)
		if err != nil {
			return nil, err
		}
		if !hasDriver {
			return nil, ErrNotHandled
		}
		return nil, unsupportedOverlayError(policy.Overlay)
	}
	// An overlay changes the Go command's effective module graph and may
	// introduce class markers or driver metadata absent from the physical tree.
	if overlay := policy.Overlay; overlay != "" {
		overlayGraph, err := loadEffectiveGraph(ctx, projectDir, policy)
		if err != nil {
			if errors.Is(err, errNoGoModule) {
				return nil, ErrNotHandled
			}
			return nil, err
		}
		hasDriver, err := overlayDriverProjectMatch(projectDir, overlayGraph, recursive)
		if err != nil {
			return nil, err
		}
		if !hasDriver {
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
		hasDriver, err := r.probeVendorProject(projectDir, preflightModule, recursive)
		if err != nil {
			return nil, err
		}
		if !hasDriver {
			return nil, ErrNotHandled
		}
		return nil, vendorUnsupportedError(string(r.policy.graph.ModMode))
	}
	return loadEffectiveGraph(ctx, projectDir, r.policy.graph)
}
