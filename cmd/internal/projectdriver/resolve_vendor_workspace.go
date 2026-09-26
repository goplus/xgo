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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gomodfile "golang.org/x/mod/modfile"
)

type workspaceVendorMember struct {
	modulePath string
	root       string
	goMod      string
}

// probeVendorWorkspacePackage handles XGo-only packages in vendor mode.
// Only go.work use members are eligible.
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

	loaded, err := loadDriverModule(selected.goMod, filepath.Join(selected.root, "gox.mod"))
	if err != nil {
		return false, fmt.Errorf("load workspace member %q metadata: %w", selected.modulePath, err)
	}
	if loaded.Path() != selected.modulePath {
		return false, fmt.Errorf("workspace member %q module path changed to %q during driver discovery", selected.modulePath, loaded.Path())
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
	ownerGoMod, err := goModForDir(ctx, r.policy.graph, projectDir)
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
