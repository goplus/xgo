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

// preflightClassMetadataDetails reads target metadata without loading the graph.
// This keeps legacy vendor targets on the existing path.
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
		if runtimeProjectMatches(projects, entry.Name()) {
			return true, nil
		}
	}
	return false, nil
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

// vendorManifestIsForWorkspace mirrors cmd/go's workspace marker.
// A missing modules.txt is treated as module vendor mode.
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
