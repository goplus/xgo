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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/module"
)

// versionedPackageHasRuntime classifies the requested version in an isolated graph.
// Caller graph metadata is deliberately not reused.
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
		// Continue with the structured probe; XGo-only packages may not be Go packages.
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
		// `go list -find` already supplied the authoritative source fields.
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
	module, hasClass, err := loadResolvedClasses(graph)
	if err != nil {
		return false, fmt.Errorf("load versioned package %q runtime metadata: %w", target, err)
	}
	if !hasClass {
		return false, nil
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
