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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type goListPackage struct {
	Dir        string
	ImportPath string
	Name       string
	Module     *goListModule
	Error      *struct {
		Err string
	}
}

func resolvePackageDirectory(ctx context.Context, graph *effectiveGraph, importPath, workDir string, policy GraphPolicy) (string, ResolvedModule, error) {
	pkg, err := listPackageTarget(ctx, importPath, workDir, policy)
	if err != nil {
		return "", ResolvedModule{}, err
	}
	if pkg.ImportPath != importPath {
		return "", ResolvedModule{}, fmt.Errorf("package target %q resolved as %q", importPath, pkg.ImportPath)
	}
	if pkg.Dir == "" || pkg.Module == nil {
		// XGo-only packages may omit physical fields; resolve them from the graph.
		return resolveXGoOnlyPackageDirectory(ctx, graph, importPath, policy)
	}
	listed, err := normalizeListedModule(*pkg.Module)
	if err != nil {
		return "", ResolvedModule{}, fmt.Errorf("package target %q: %w", importPath, err)
	}
	module, ok := graph.Modules[listed.Selected.Path]
	if !ok {
		return "", ResolvedModule{}, fmt.Errorf("package target %q does not match the effective module graph", importPath)
	}
	dir, err := graph.files.canonicalDir(pkg.Dir)
	if err != nil {
		return "", ResolvedModule{}, fmt.Errorf("package target %q: %w", importPath, err)
	}
	if module.Effective().Dir == "" {
		// Keep unmarked dependency identity without materializing its source.
		return dir, module, nil
	}
	if !sameResolvedModule(listed, module) {
		return "", ResolvedModule{}, fmt.Errorf("package target %q does not match the effective module graph", importPath)
	}
	if err := validatePackagePath(module, importPath, dir); err != nil {
		return "", ResolvedModule{}, err
	}
	return dir, module, nil
}

func validatePackagePath(module ResolvedModule, importPath, dir string) error {
	if !pathWithin(module.Effective().Dir, dir) {
		return fmt.Errorf("package target %q escapes module %q", importPath, module.Selected.Path)
	}
	return nil
}

func validatePackageOwnership(ctx context.Context, policy GraphPolicy, module ResolvedModule, importPath, dir string) error {
	ownerGoMod, err := goEnvWithPolicy(ctx, policy, dir, "GOMOD")
	if err != nil {
		return fmt.Errorf("resolve package target %q module ownership: %w", importPath, err)
	}
	if ownerGoMod == "" || ownerGoMod == os.DevNull {
		return fmt.Errorf("package target %q has no module ownership", importPath)
	}
	ownerRoot, err := canonicalExistingDir(filepath.Dir(ownerGoMod))
	if err != nil {
		return fmt.Errorf("resolve package target %q module ownership: %w", importPath, err)
	}
	same, err := sameFile(ownerRoot, module.Effective().Dir)
	if err != nil {
		return fmt.Errorf("resolve package target %q module ownership: %w", importPath, err)
	}
	if !same {
		return fmt.Errorf("package target %q crosses a nested module boundary", importPath)
	}
	return nil
}

func moduleOwnsPackage(ctx context.Context, policy GraphPolicy, moduleGoMod, dir string) (bool, error) {
	ownerGoMod, err := goEnvWithPolicy(ctx, policy, dir, "GOMOD")
	if err != nil {
		return false, err
	}
	if ownerGoMod == "" || ownerGoMod == os.DevNull {
		return false, nil
	}
	return sameFile(moduleGoMod, ownerGoMod)
}

func listPackageTarget(ctx context.Context, importPath, workDir string, policy GraphPolicy) (goListPackage, error) {
	if importPath == "" || strings.Contains(importPath, "@") || strings.Contains(importPath, "...") {
		return goListPackage{}, fmt.Errorf("runtime provider does not support package target %q", importPath)
	}
	stdout, _, err := runGraphCommand(ctx, policy, workDir, "resolve package target "+importPath, policy.goArgs("list", "-e", "-find", "-json", importPath)...)
	if err != nil {
		return goListPackage{}, err
	}
	dec := json.NewDecoder(bytes.NewReader(stdout))
	var pkg goListPackage
	if err := dec.Decode(&pkg); err != nil {
		return goListPackage{}, fmt.Errorf("decode package target %q: %w", importPath, err)
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return goListPackage{}, fmt.Errorf("package target %q resolved to multiple packages", importPath)
		}
		return goListPackage{}, fmt.Errorf("decode package target %q: %w", importPath, err)
	}
	return pkg, nil
}

func resolveXGoOnlyPackageDirectory(ctx context.Context, graph *effectiveGraph, importPath string, policy GraphPolicy) (string, ResolvedModule, error) {
	paths := make([]string, 0, len(graph.Modules))
	for path := range graph.Modules {
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool { return len(paths[i]) > len(paths[j]) })
	for _, modulePath := range paths {
		if !moduleContainsPackage(modulePath, importPath) {
			continue
		}
		module := graph.Modules[modulePath]
		root := module.Effective().Dir
		if root == "" {
			return "", ResolvedModule{}, fmt.Errorf("package target %q has no materialized module source", importPath)
		}
		suffix := strings.TrimPrefix(importPath, modulePath)
		candidate := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(suffix, "/")))
		dir, err := graph.files.canonicalDir(candidate)
		if err != nil {
			return "", ResolvedModule{}, fmt.Errorf("package target %q: %w", importPath, err)
		}
		if err := validatePackagePath(module, importPath, dir); err != nil {
			return "", ResolvedModule{}, err
		}
		// Overlay-only packages have no physical cwd; the graph owns their identity.
		if graph.files != nil && graph.files.hasReplacementChild(dir) {
			return dir, module, nil
		}
		if err := validatePackageOwnership(ctx, policy, module, importPath, dir); err != nil {
			return "", ResolvedModule{}, err
		}
		return dir, module, nil
	}
	return "", ResolvedModule{}, fmt.Errorf("package target %q is outside the effective module graph", importPath)
}

// retargetEffectiveGraph keeps the caller graph and changes only its target module.
func retargetEffectiveGraph(graph *effectiveGraph, target ResolvedModule) (*effectiveGraph, error) {
	modfilePath := target.Effective().GoMod
	if target.Selected.Path == graph.Target.Selected.Path {
		modfilePath = graph.TargetModFile.Path
	}
	identity, classPaths, err := readTargetModFileView(modfilePath, graph.files)
	if err != nil {
		return nil, err
	}
	classModules := make([]ResolvedModule, 0, len(classPaths))
	for _, path := range classPaths {
		module, ok := graph.Modules[path]
		if !ok {
			return nil, fmt.Errorf("class module %q is absent from the effective graph", path)
		}
		if module.Effective().Dir == "" || module.Effective().GoMod == "" {
			return nil, fmt.Errorf("class module %q has no materialized effective source", path)
		}
		classModules = append(classModules, module)
	}
	return &effectiveGraph{
		Target:        target,
		Modules:       graph.Modules,
		ClassModules:  classModules,
		TargetModFile: identity,
		files:         graph.files,
	}, nil
}

func moduleContainsPackage(modulePath, packagePath string) bool {
	return packagePath == modulePath || strings.HasPrefix(packagePath, modulePath+"/")
}
