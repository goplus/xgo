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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/goplus/mod/modfile"
	"github.com/goplus/mod/modload"
	gomodfile "golang.org/x/mod/modfile"
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
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !pathWithin(root, projectDir) {
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

func (r *Resolver) vendorClassMetadataError(modulePath string) error {
	return fmt.Errorf("%w: class module %q metadata is not represented by standard Go vendor data", vendorUnsupportedError(string(r.policy.graph.ModMode)), modulePath)
}

func (r *Resolver) matchVendorModule(projectDir string, module modload.Module, recursive bool) (bool, error) {
	if classPath := externalClassModule(module); classPath != "" {
		return false, r.vendorClassMetadataError(classPath)
	}
	return matchVendorProjects(projectDir, module.Projects(), recursive)
}

func matchVendorProjects(dir string, projects []*modfile.Project, recursive bool) (bool, error) {
	if recursive {
		return patternContainsDriverProjects(dir, projects)
	}
	return hasDriverProject(dir, projects)
}

func hasDriverProject(dir string, projects []*modfile.Project) (bool, error) {
	if len(projects) == 0 {
		return false, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if !driverProjectMatches(projects, entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return false, err
		}
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("driver project file %q is not a regular non-symlink file", filepath.Join(dir, entry.Name()))
		}
		return true, nil
	}
	return false, nil
}

func effectiveVendorMode(policy GraphPolicy, moduleGoMod string, parsed *gomodfile.File) (bool, error) {
	if policy.ModMode != "" {
		return policy.ModMode == modModeVendor, nil
	}
	workspace := policy.GoWork != "" && policy.GoWork != "off"
	var goVersion, vendorDir string
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

// A missing modules.txt denotes module vendor mode.
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
	return fmt.Errorf("%w (%s); select -mod=readonly or -mod=mod explicitly", ErrDriverVendorUnsupported, mode)
}
