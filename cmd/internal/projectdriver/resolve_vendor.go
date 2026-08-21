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

	"github.com/goplus/mod/modload"
)

// preflightClassMetadata reads target metadata without loading the graph.
// This keeps legacy vendor targets on the existing path.
func (r *Resolver) preflightClassMetadata(ctx context.Context, dir string) (loaded modload.Module, moduleGoMod string, hasClass, vendor bool, err error) {
	goMod, err := goModForDir(ctx, r.policy.graph, dir)
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
	loaded, err = loadDriverModule(identity.Path, filepath.Join(filepath.Dir(moduleGoMod), "gox.mod"))
	if err != nil {
		return loaded, moduleGoMod, false, false, err
	}
	hasClass = len(classMods) != 0 || loaded.HasProject()
	vendor, err = effectiveVendorMode(r.policy.graph, moduleGoMod, loaded.File)
	return loaded, moduleGoMod, hasClass, vendor, err
}

// probeVendorProject classifies a target without go list -m all.
// External class metadata is unavailable in standard vendor snapshots.
func (r *Resolver) probeVendorProject(projectDir string, target modload.Module, recursive bool) (bool, error) {
	return r.matchVendorModule(projectDir, target, recursive)
}

// probeVendorPackage uses package-specific go list, which works in vendor mode.
// Only main/workspace metadata is authoritative.
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
		// An unmarked dependency cannot expand the driver trust boundary.
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
	loaded, err := loadDriverModule(module.Effective().GoMod, filepath.Join(root, "gox.mod"))
	if err != nil {
		return false, err
	}
	return r.matchVendorModule(projectDir, loaded, recursive)
}

func (r *Resolver) probeVendorPackageInModule(ctx context.Context, moduleGoMod string, target modload.Module, importPath string, recursive bool) (bool, error) {
	root := filepath.Dir(moduleGoMod)
	suffix := strings.TrimPrefix(importPath, target.Path())
	dir := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(suffix, "/")))
	projectDir, err := canonicalExistingDir(dir)
	if err != nil || !pathWithin(root, projectDir) {
		return false, nil
	}
	same, err := moduleOwnsPackage(ctx, r.policy.graph, moduleGoMod, projectDir)
	if err != nil {
		return false, err
	}
	if !same {
		return false, nil
	}
	return matchVendorProjects(projectDir, target.Projects(), recursive)
}
