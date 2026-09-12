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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/goplus/mod/modload"
	gomodfile "golang.org/x/mod/modfile"
)

var errNoGoModule = errors.New("driver discovery requires a Go module")

type fileIdentity = modload.FileIdentity

type effectiveGraph struct {
	Target        ResolvedModule
	Modules       map[string]ResolvedModule
	ClassModules  []ResolvedModule
	TargetModFile fileIdentity
	files         *graphFileView
}

type goListModule struct {
	Path    string
	Version string
	Replace *goListModule
	Main    bool
	Dir     string
	GoMod   string
	Error   *struct {
		Err string
	}
}

func loadEffectiveGraph(ctx context.Context, projectDir string, policy GraphPolicy) (*effectiveGraph, error) {
	view, err := newGraphFileView(policy, projectDir)
	if err != nil {
		return nil, err
	}
	stdout, stderr, err := runGraphCommand(ctx, policy, projectDir, "go list -m", policy.goArgs("list", "-m", "-json", "all")...)
	if err != nil {
		message := string(stderr)
		if strings.Contains(message, "go.mod file not found") || strings.Contains(message, "cannot find main module") {
			return nil, fmt.Errorf("%w: %s", errNoGoModule, strings.TrimSpace(message))
		}
		return nil, err
	}
	var raw []goListModule
	dec := json.NewDecoder(bytes.NewReader(stdout))
	for {
		var module goListModule
		if err := dec.Decode(&module); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decode effective module graph: %w", err)
		}
		raw = append(raw, module)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("effective module graph is empty")
	}
	projectDir, err = canonicalExistingDir(projectDir)
	if err != nil {
		return nil, err
	}
	ret := &effectiveGraph{Modules: make(map[string]ResolvedModule, len(raw))}
	for _, module := range raw {
		resolved, err := normalizeGraphModule(module)
		if err != nil {
			return nil, err
		}
		if _, exists := ret.Modules[resolved.Selected.Path]; exists {
			return nil, fmt.Errorf("duplicate logical module %q in effective graph", resolved.Selected.Path)
		}
		ret.Modules[resolved.Selected.Path] = resolved
	}
	ret.Target, err = selectTargetModule(projectDir, ret.Modules)
	if err != nil {
		return nil, err
	}
	modfilePath := ret.Target.Effective().GoMod
	if policy.ModFile != "" {
		modfilePath = policy.ModFile
	}
	identity, classPaths, err := readTargetModFileView(modfilePath, view)
	if err != nil {
		return nil, err
	}
	classModules, err := materializeClassModules(ctx, projectDir, policy, ret.Modules, classPaths)
	if err != nil {
		return nil, err
	}
	ret.ClassModules = classModules
	ret.TargetModFile = identity
	ret.files = view
	return ret, nil
}

func materializeClassModules(ctx context.Context, dir string, policy GraphPolicy, modules map[string]ResolvedModule, paths []string) ([]ResolvedModule, error) {
	resolved := make([]ResolvedModule, 0, len(paths))
	for _, path := range paths {
		module, ok := modules[path]
		if !ok {
			return nil, fmt.Errorf("class module %q is absent from the effective graph", path)
		}
		if module.Effective().Dir == "" || module.Effective().GoMod == "" {
			var err error
			module, err = downloadGraphModule(ctx, dir, policy, module)
			if err != nil {
				return nil, fmt.Errorf("materialize class module %q: %w", path, err)
			}
			modules[path] = module
		}
		resolved = append(resolved, module)
	}
	return resolved, nil
}

func selectTargetModule(projectDir string, modules map[string]ResolvedModule) (ResolvedModule, error) {
	bestRoot := ""
	candidates := make([]ResolvedModule, 0, 1)
	for _, module := range modules {
		root := module.Effective().Dir
		if root == "" || !pathWithin(root, projectDir) {
			continue
		}
		switch {
		case len(root) > len(bestRoot):
			bestRoot = root
			candidates = append(candidates[:0], module)
		case len(root) == len(bestRoot):
			candidates = append(candidates, module)
		}
	}
	if len(candidates) == 0 {
		return ResolvedModule{}, fmt.Errorf("project directory %q is outside the effective module graph", projectDir)
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	paths := make([]string, len(candidates))
	for i, candidate := range candidates {
		paths[i] = candidate.Selected.Path
	}
	sort.Strings(paths)
	return ResolvedModule{}, fmt.Errorf("project directory %q has ambiguous effective module owners: %s", projectDir, strings.Join(paths, ", "))
}

type goDownloadModule struct {
	Path    string
	Version string
	Dir     string
	GoMod   string
	Error   string
}

func downloadGraphModule(ctx context.Context, dir string, policy GraphPolicy, resolved ResolvedModule) (ResolvedModule, error) {
	effective := resolved.Effective()
	if effective.Version == "" {
		return ResolvedModule{}, fmt.Errorf("local effective source is missing")
	}
	query := effective.Path + "@" + effective.Version
	stdout, _, err := runGraphCommand(ctx, policy, dir, "go mod download "+query, "mod", "download", "-json", query)
	if err != nil {
		return ResolvedModule{}, err
	}
	var downloaded goDownloadModule
	if err := json.Unmarshal(stdout, &downloaded); err != nil {
		return ResolvedModule{}, fmt.Errorf("decode downloaded module: %w", err)
	}
	if downloaded.Error != "" {
		return ResolvedModule{}, errors.New(downloaded.Error)
	}
	if downloaded.Path != effective.Path || downloaded.Version != effective.Version {
		return ResolvedModule{}, fmt.Errorf("downloaded module identity %s@%s does not match %s", downloaded.Path, downloaded.Version, query)
	}
	sourceDir, goMod, err := canonicalModuleSource(effective.Path, downloaded.Dir, downloaded.GoMod)
	if err != nil {
		return ResolvedModule{}, err
	}
	if resolved.Replace == nil {
		resolved.Selected.Dir, resolved.Selected.GoMod = sourceDir, goMod
	} else {
		resolved.Replace.Dir, resolved.Replace.GoMod = sourceDir, goMod
	}
	if err := resolved.Validate(); err != nil {
		return ResolvedModule{}, fmt.Errorf("downloaded module %q: %w", query, err)
	}
	return resolved, nil
}

func normalizeListedModule(module goListModule) (ResolvedModule, error) {
	return normalizeModule(module, true)
}

func normalizeGraphModule(module goListModule) (ResolvedModule, error) {
	return normalizeModule(module, false)
}

func normalizeModule(module goListModule, requireSource bool) (ResolvedModule, error) {
	if module.Error != nil && module.Error.Err != "" {
		return ResolvedModule{}, fmt.Errorf("module %q: %s", module.Path, module.Error.Err)
	}
	if module.Path == "" {
		return ResolvedModule{}, fmt.Errorf("effective graph contains a module with no path")
	}
	ret := ResolvedModule{
		Selected: ModuleRef{Path: module.Path, Version: module.Version},
		Main:     module.Main,
	}
	if module.Replace == nil {
		if !requireSource && (module.Dir == "" || module.GoMod == "") {
			return ret, nil
		}
		dir, goMod, err := canonicalModuleSource(module.Path, module.Dir, module.GoMod)
		if err != nil {
			return ResolvedModule{}, err
		}
		ret.Selected.Dir, ret.Selected.GoMod = dir, goMod
		if err := ret.Validate(); err != nil {
			return ResolvedModule{}, fmt.Errorf("module %q: %w", module.Path, err)
		}
		return ret, nil
	}
	if !requireSource && (module.Replace.Dir == "" || module.Replace.GoMod == "") {
		replacePath := module.Replace.Path
		ret.Replace = &ModuleRef{Path: replacePath, Version: module.Replace.Version}
		return ret, nil
	}
	dir, goMod, err := canonicalModuleSource(module.Replace.Path, module.Replace.Dir, module.Replace.GoMod)
	if err != nil {
		return ResolvedModule{}, err
	}
	replacePath := module.Replace.Path
	if module.Replace.Version == "" {
		// go list preserves the spelling from the replace directive (often
		// ../framework). The resolved graph contract carries the canonical
		// filesystem identity for a local replacement.
		replacePath = dir
	}
	ret.Replace = &ModuleRef{
		Path:    replacePath,
		Version: module.Replace.Version,
		Dir:     dir,
		GoMod:   goMod,
	}
	if err := ret.Validate(); err != nil {
		return ResolvedModule{}, fmt.Errorf("module %q: %w", module.Path, err)
	}
	return ret, nil
}

func canonicalModuleSource(path, dir, goMod string) (string, string, error) {
	if dir == "" || goMod == "" {
		return "", "", fmt.Errorf("effective source for module %q is incomplete (vendor mode is unsupported for class projects)", path)
	}
	canonicalDir, err := canonicalExistingDir(dir)
	if err != nil {
		return "", "", fmt.Errorf("module %q directory: %w", path, err)
	}
	canonicalGoMod, err := canonicalExistingFile(goMod)
	if err != nil {
		return "", "", fmt.Errorf("module %q go.mod: %w", path, err)
	}
	return canonicalDir, canonicalGoMod, nil
}

func readTargetModFile(path string) (fileIdentity, []string, error) {
	return readTargetModFileView(path, nil)
}

func readTargetModFileView(path string, view *graphFileView) (fileIdentity, []string, error) {
	canonical, err := view.canonicalFile(path)
	if err != nil {
		return fileIdentity{}, nil, fmt.Errorf("effective target modfile: %w", err)
	}
	data, err := view.readFile(canonical)
	if err != nil {
		return fileIdentity{}, nil, fmt.Errorf("read effective target modfile: %w", err)
	}
	parsed, err := gomodfile.Parse(canonical, data, nil)
	if err != nil {
		return fileIdentity{}, nil, err
	}
	classMods := make([]string, 0)
	for _, require := range parsed.Require {
		if require.Syntax == nil || !modload.HasClassMarker(require.Syntax.Suffix) {
			continue
		}
		classMods = append(classMods, require.Mod.Path)
	}
	return fileIdentity{Path: canonical, SHA256: sha256Bytes(data)}, classMods, nil
}
