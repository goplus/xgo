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
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goplus/mod/modload"
	gomodfile "golang.org/x/mod/modfile"
)

var errNoGoModule = errors.New("runtime discovery requires a Go module")

type fileIdentity = modload.FileIdentity

type effectiveGraph struct {
	Target        ResolvedModule
	Modules       map[string]ResolvedModule
	ClassModules  []ResolvedModule
	TargetModFile fileIdentity
	files         *graphFileView
}

// graphFileView keeps the logical paths changed by -overlay. Runtime-provider
// v1 never executes against this view: it uses the map to classify project
// files, while the Go command remains authoritative for graph resolution.
type graphFileView struct {
	workDir      string
	replacements map[string]string
}

type overlayJSON struct {
	Replace map[string]string
}

func newGraphFileView(policy GraphPolicy, workDir string) (*graphFileView, error) {
	view := &graphFileView{workDir: workDir}
	overlay := policy.Overlay
	if overlay == "" {
		return view, nil
	}
	data, err := os.ReadFile(overlay)
	if err != nil {
		return nil, fmt.Errorf("read overlay %q: %w", overlay, err)
	}
	var parsed overlayJSON
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parse overlay %q: %w", overlay, err)
	}
	view.replacements = make(map[string]string, len(parsed.Replace))
	for from, to := range parsed.Replace {
		if from == "" {
			return nil, fmt.Errorf("overlay %q contains an empty replacement path", overlay)
		}
		from = overlayPath(workDir, from)
		if _, duplicate := view.replacements[from]; duplicate {
			return nil, fmt.Errorf("overlay %q contains duplicate normalized path %q", overlay, from)
		}
		view.replacements[from] = overlayPath(workDir, to)
	}
	for parent, target := range view.replacements {
		if target == "" {
			continue
		}
		for child, childTarget := range view.replacements {
			if childTarget != "" && child != parent && pathWithin(parent, child) {
				return nil, fmt.Errorf("overlay %q maps both file %q and child %q", overlay, parent, child)
			}
		}
	}
	return view, nil
}

func overlayPath(workDir, path string) string {
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}
	return filepath.Clean(path)
}

func (v *graphFileView) hasOverlay() bool {
	return v != nil && v.replacements != nil
}

func (v *graphFileView) readFile(path string) ([]byte, error) {
	if v == nil || !v.hasOverlay() {
		return os.ReadFile(path)
	}
	logical := overlayPath(v.workDir, path)
	if actual, ok := v.replacements[logical]; ok {
		if actual == "" {
			return nil, &os.PathError{Op: "read", Path: logical, Err: os.ErrNotExist}
		}
		return os.ReadFile(actual)
	}
	// An exact child replacement wins over an ancestor deletion, as in cmd/go.
	if v.hasReplacementAncestor(logical) {
		return nil, &os.PathError{Op: "read", Path: logical, Err: os.ErrNotExist}
	}
	return os.ReadFile(logical)
}

// hasReplacementAncestor reports whether a path is below an exact overlay
// entry. Both a deleted parent and a parent replaced by a regular file hide
// children. Callers must check an exact entry first, because an exact child
// replacement takes precedence over a deleted parent.
func (v *graphFileView) hasReplacementAncestor(path string) bool {
	_, ok := v.replacementAncestor(path)
	return ok
}

// replacementAncestor returns the nearest exact overlay entry above path.
// Nearest-entry selection matches cmd/go's path-prefix lookup and matters for
// nested deletion/addition overlays.
func (v *graphFileView) replacementAncestor(path string) (string, bool) {
	if v == nil || !v.hasOverlay() {
		return "", false
	}
	for parent := filepath.Dir(path); parent != path; parent = filepath.Dir(parent) {
		if actual, ok := v.replacements[parent]; ok {
			return actual, true
		}
		if parent == filepath.Dir(parent) {
			break
		}
	}
	return "", false
}

// hasReplacementBelow reports whether a non-deleted overlay key is below path.
// A deletion does not synthesize a directory and must not hide a physical file.
func (v *graphFileView) hasReplacementBelow(path string) bool {
	if v == nil || !v.hasOverlay() {
		return false
	}
	for candidate, actual := range v.replacements {
		if actual != "" && candidate != path && pathWithin(path, candidate) {
			return true
		}
	}
	return false
}

// hasVirtualDirectory reports whether an overlay child implies path as a
// directory. The replacement destination is intentionally opaque: Go follows
// it when it reads a file, and classification must not reject symlinks,
// missing files, or directories before a runtime match is known.
func (v *graphFileView) hasVirtualDirectory(path string) bool {
	if v == nil || !v.hasOverlay() {
		return false
	}
	for logical, actual := range v.replacements {
		if actual != "" && logical != path && pathWithin(path, logical) {
			return true
		}
	}
	return false
}

func (v *graphFileView) directoryVisible(dir string) (bool, error) {
	if v == nil || !v.hasOverlay() {
		info, err := os.Stat(dir)
		if os.IsNotExist(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return info.IsDir(), nil
	}
	logical := overlayPath(v.workDir, dir)
	if _, exact := v.replacements[logical]; exact {
		return false, nil
	}
	ancestor, hasAncestor := v.replacementAncestor(logical)
	if hasAncestor && ancestor != "" {
		return false, nil
	}
	if v.hasVirtualDirectory(logical) {
		return true, nil
	}
	info, err := os.Stat(logical)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func (v *graphFileView) physicalEntries(dir string) ([]os.DirEntry, error) {
	if v == nil || !v.hasOverlay() {
		return os.ReadDir(dir)
	}
	logical := overlayPath(v.workDir, dir)
	if _, exact := v.replacements[logical]; exact {
		return nil, nil
	}
	if _, hasAncestor := v.replacementAncestor(logical); hasAncestor {
		return nil, nil
	}
	entries, err := os.ReadDir(logical)
	if os.IsNotExist(err) && v.hasVirtualDirectory(logical) {
		return nil, nil
	}
	return entries, err
}

// regularFileNames merges physical entries with overlay keys, but never stats
// a replacement destination. This keeps ordinary legacy overlays unchanged.
func (v *graphFileView) regularFileNames(dir string) ([]string, error) {
	logicalDir := dir
	if v != nil && v.hasOverlay() {
		logicalDir = overlayPath(v.workDir, dir)
	}
	entries, err := v.physicalEntries(logicalDir)
	if err != nil {
		return nil, err
	}
	files := make(map[string]bool, len(entries))
	for _, entry := range entries {
		logical := filepath.Join(logicalDir, entry.Name())
		if v != nil && v.hasOverlay() {
			if actual, replaced := v.replacements[logical]; replaced {
				files[entry.Name()] = actual != ""
				continue
			}
			if v.hasReplacementBelow(logical) {
				continue
			}
		}
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil, infoErr
		}
		if info.Mode().IsRegular() {
			files[entry.Name()] = true
		}
	}
	if v != nil && v.hasOverlay() {
		for logical, actual := range v.replacements {
			if filepath.Dir(logical) != logicalDir {
				continue
			}
			files[filepath.Base(logical)] = actual != ""
		}
	}
	names := make([]string, 0, len(files))
	for name, visible := range files {
		if visible {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

// directoryNames returns physical child directories plus the first component
// of each non-deleted overlay key. It is used only by pattern classification.
func (v *graphFileView) directoryNames(dir string) ([]string, error) {
	logicalDir := overlayPath(v.workDir, dir)
	visible, err := v.directoryVisible(logicalDir)
	if err != nil {
		return nil, err
	}
	if !visible {
		return nil, nil
	}
	entries, err := v.physicalEntries(logicalDir)
	if err != nil {
		return nil, err
	}
	direct := make(map[string]bool, len(entries))
	for _, entry := range entries {
		logical := filepath.Join(logicalDir, entry.Name())
		if actual, replaced := v.replacements[logical]; replaced {
			direct[entry.Name()] = false
			if actual == "" {
				continue
			}
			continue
		}
		if v.hasReplacementBelow(logical) {
			direct[entry.Name()] = true
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil, infoErr
		}
		if info.IsDir() {
			direct[entry.Name()] = true
		}
	}
	for logical, actual := range v.replacements {
		if actual == "" || !pathWithin(logicalDir, logical) {
			continue
		}
		rel, relErr := filepath.Rel(logicalDir, logical)
		if relErr != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		name := rel
		if at := strings.IndexByte(rel, filepath.Separator); at >= 0 {
			name = rel[:at]
		}
		child := filepath.Join(logicalDir, name)
		if childActual, exact := v.replacements[child]; exact {
			direct[name] = childActual == ""
			continue
		}
		direct[name] = true
	}
	names := make([]string, 0, len(direct))
	for name, visible := range direct {
		if visible {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func (v *graphFileView) regularFileVisible(path string) (bool, error) {
	if v == nil || !v.hasOverlay() {
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return info.Mode().IsRegular(), nil
	}
	logical := overlayPath(v.workDir, path)
	if actual, exact := v.replacements[logical]; exact {
		if actual == "" {
			return false, nil
		}
		return true, nil
	}
	if v.hasReplacementAncestor(logical) {
		return false, nil
	}
	info, err := os.Stat(logical)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.Mode().IsRegular(), nil
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

type goListPackage struct {
	Dir        string
	ImportPath string
	Name       string
	Module     *goListModule
	Error      *struct {
		Err string
	}
}

func preparePolicies(ctx context.Context, cwd string, cli []string) (parsedFlags, error) {
	goCommand, err := hostGoCommand()
	if err != nil {
		return parsedFlags{}, err
	}
	ambient, err := goEnvValue(ctx, goCommand, cwd, "GOFLAGS", false)
	if err != nil {
		return parsedFlags{}, err
	}
	policy, err := parseRuntimeFlags(cwd, goCommand, "off", ambient, cli)
	if err != nil {
		return parsedFlags{}, err
	}
	policy, err = sanitizeGraphFlags(policy)
	if err != nil {
		return parsedFlags{}, err
	}
	// GOWORK does not depend on -mod/-modfile/-overlay. Keep graph flags out of
	// GOFLAGS entirely: their canonical paths are passed as distinct argv
	// elements to every graph command, which also preserves spaces and Windows
	// path separators without a second quoting grammar.
	goWork, err := goEnvValue(ctx, goCommand, cwd, "GOWORK", true)
	if err != nil {
		return parsedFlags{}, err
	}
	if goWork == "" || goWork == "off" {
		policy.graph.GoWork = "off"
	} else {
		goWork, err = canonicalExistingFile(goWork)
		if err != nil {
			return parsedFlags{}, fmt.Errorf("effective go.work: %w", err)
		}
		policy.graph.GoWork = goWork
	}
	return policy, nil
}

// sanitizeGraphFlags removes graph files that do not exist from discovery.
// They remain rejected runtime flags, so a matched runtime still reports the
// policy error through BuildPolicy while a legacy target can continue with its
// normal command path.
func sanitizeGraphFlags(policy parsedFlags) (parsedFlags, error) {
	if path := policy.graph.ModFile; path != "" {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			policy.rejected = append(policy.rejected, "-modfile="+path)
			policy.graph.ModFile = ""
		} else if err != nil {
			return parsedFlags{}, fmt.Errorf("inspect -modfile input %q: %w", path, err)
		}
	}
	if path := policy.graph.Overlay; path != "" {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			policy.rejected = append(policy.rejected, "-overlay="+path)
			policy.graph.Overlay = ""
		} else if err != nil {
			return parsedFlags{}, fmt.Errorf("inspect -overlay input %q: %w", path, err)
		}
	}
	return policy, nil
}

func hostGoCommand() (string, error) {
	path, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("host Go command: %w", err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("host Go command %q is not a regular file", path)
	}
	return path, nil
}

func goEnvValue(ctx context.Context, goCommand, dir, key string, clearGOFLAGS bool) (string, error) {
	cmd := commandContext(ctx, goCommand, "env", key)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	// GOFLAGS itself must be read from the ambient environment; GOWORK is
	// queried with GOFLAGS cleared so an ambient graph flag cannot affect it.
	if clearGOFLAGS {
		cmd.Env = replaceEnv(cmd.Env, "GOFLAGS", "")
	}
	out, err := cmd.Output()
	if err != nil {
		return "", commandError("go env "+key, err, string(cmdStderr(cmd)))
	}
	return strings.TrimSpace(string(out)), nil
}

func loadEffectiveGraph(ctx context.Context, projectDir string, policy GraphPolicy) (*effectiveGraph, error) {
	view, err := newGraphFileView(policy, projectDir)
	if err != nil {
		return nil, err
	}
	args := policy.goArgs("list", "-m", "-json", "all")
	cmd := exec.CommandContext(ctx, policy.GoCommand, args...)
	cmd.Dir = projectDir
	cmd.Env = graphEnvironment(os.Environ(), policy.GoWork)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := stderr.String()
		if strings.Contains(message, "go.mod file not found") || strings.Contains(message, "cannot find main module") {
			return nil, fmt.Errorf("%w: %s", errNoGoModule, strings.TrimSpace(message))
		}
		return nil, commandError("go list -m", err, message)
	}
	var raw []goListModule
	dec := json.NewDecoder(&stdout)
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
	bestRoot := ""
	for _, module := range raw {
		resolved, err := normalizeGraphModule(module)
		if err != nil {
			return nil, err
		}
		if _, exists := ret.Modules[resolved.Selected.Path]; exists {
			return nil, fmt.Errorf("duplicate logical module %q in effective graph", resolved.Selected.Path)
		}
		ret.Modules[resolved.Selected.Path] = resolved
		effective := resolved.Effective()
		if effective.Dir != "" && pathWithin(effective.Dir, projectDir) && len(effective.Dir) > len(bestRoot) {
			bestRoot = effective.Dir
			ret.Target = resolved
		}
	}
	if bestRoot == "" {
		return nil, fmt.Errorf("project directory %q is outside the effective module graph", projectDir)
	}
	modfilePath := ret.Target.Effective().GoMod
	if policy.ModFile != "" {
		modfilePath = policy.ModFile
	}
	identity, classPaths, err := readTargetModFileView(modfilePath, view)
	if err != nil {
		return nil, err
	}
	classModules := make([]ResolvedModule, 0, len(classPaths))
	for _, path := range classPaths {
		module, ok := ret.Modules[path]
		if !ok {
			return nil, fmt.Errorf("class module %q is absent from the effective graph", path)
		}
		if module.Effective().Dir == "" || module.Effective().GoMod == "" {
			module, err = downloadGraphModule(ctx, projectDir, policy, module)
			if err != nil {
				return nil, fmt.Errorf("materialize class module %q: %w", path, err)
			}
			ret.Modules[path] = module
		}
		classModules = append(classModules, module)
	}
	ret.ClassModules = classModules
	ret.TargetModFile = identity
	ret.files = view
	return ret, nil
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
	cmd := exec.CommandContext(ctx, policy.GoCommand, "mod", "download", "-json", query)
	cmd.Dir = dir
	cmd.Env = graphEnvironment(os.Environ(), policy.GoWork)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return ResolvedModule{}, commandError("go mod download "+query, err, stderr.String())
	}
	var downloaded goDownloadModule
	if err := json.Unmarshal(stdout.Bytes(), &downloaded); err != nil {
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

// canonicalFile pins a physical file in the usual case, but keeps the logical
// path for an overlay-only file. The latter is necessary because cmd/go can
// create go.mod (or an alternate modfile) solely through -overlay.
func (v *graphFileView) canonicalFile(path string) (string, error) {
	if v == nil || !v.hasOverlay() {
		return canonicalExistingFile(path)
	}
	logical := overlayPath(v.workDir, path)
	if actual, exact := v.replacements[logical]; exact {
		if actual == "" {
			return "", &os.PathError{Op: "stat", Path: logical, Err: os.ErrNotExist}
		}
		return logical, nil
	}
	if v.hasReplacementAncestor(logical) {
		return "", &os.PathError{Op: "stat", Path: logical, Err: os.ErrNotExist}
	}
	return canonicalExistingFile(logical)
}

// canonicalDir is the directory counterpart to canonicalFile. A synthetic
// directory has no filesystem inode, so its clean logical path is returned;
// this is safe for classification because provider execution is rejected for
// overlay-backed runtime targets before any path is sent over the protocol.
func (v *graphFileView) canonicalDir(path string) (string, error) {
	if v == nil || !v.hasOverlay() {
		return canonicalExistingDir(path)
	}
	logical := overlayPath(v.workDir, path)
	visible, err := v.directoryVisible(logical)
	if err != nil {
		return "", err
	}
	if !visible {
		return "", &os.PathError{Op: "stat", Path: logical, Err: os.ErrNotExist}
	}
	if v.hasVirtualDirectory(logical) {
		return logical, nil
	}
	return canonicalExistingDir(logical)
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
		// The Go command may omit physical fields for an XGo-only package.
		// Resolve that candidate from the selected graph, then ask Go which
		// module root owns it so a prefix match cannot cross a nested module.
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
		// Unmarked dependencies are not materialized during runtime discovery.
		// Return their authoritative package identity so Resolve can classify
		// them as legacy before attempting any runtime metadata access.
		return dir, module, nil
	}
	if !sameResolvedModule(listed, module) {
		return "", ResolvedModule{}, fmt.Errorf("package target %q does not match the effective module graph", importPath)
	}
	if !pathWithin(module.Effective().Dir, dir) {
		return "", ResolvedModule{}, fmt.Errorf("package target %q escapes module %q", importPath, module.Selected.Path)
	}
	return dir, module, nil
}

func listPackageTarget(ctx context.Context, importPath, workDir string, policy GraphPolicy) (goListPackage, error) {
	if importPath == "" || strings.Contains(importPath, "@") || strings.Contains(importPath, "...") {
		return goListPackage{}, fmt.Errorf("runtime provider does not support package target %q", importPath)
	}
	args := policy.goArgs("list", "-e", "-find", "-json", importPath)
	cmd := exec.CommandContext(ctx, policy.GoCommand, args...)
	cmd.Dir = workDir
	cmd.Env = graphEnvironment(os.Environ(), policy.GoWork)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return goListPackage{}, commandError("resolve package target "+importPath, err, stderr.String())
	}
	dec := json.NewDecoder(&stdout)
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
		if !pathWithin(root, dir) {
			return "", ResolvedModule{}, fmt.Errorf("package target %q escapes module %q", importPath, modulePath)
		}
		// An XGo-only package can be supplied entirely by an overlay, in which
		// case its logical directory has no physical cwd for `go env` to enter.
		// The effective graph already established the owning module; retain the
		// nested-module check for physical directories and use that graph identity
		// for synthetic ones.
		if graph.files != nil && graph.files.hasVirtualDirectory(dir) {
			return dir, module, nil
		}
		ownerGoMod, err := goEnvWithPolicy(ctx, policy, dir, "GOMOD")
		if err != nil {
			return "", ResolvedModule{}, fmt.Errorf("resolve package target %q module ownership: %w", importPath, err)
		}
		if ownerGoMod == "" || ownerGoMod == os.DevNull {
			return "", ResolvedModule{}, fmt.Errorf("package target %q has no module ownership", importPath)
		}
		ownerRoot, err := canonicalExistingDir(filepath.Dir(ownerGoMod))
		if err != nil {
			return "", ResolvedModule{}, fmt.Errorf("resolve package target %q module ownership: %w", importPath, err)
		}
		same, err := sameFile(ownerRoot, root)
		if err != nil {
			return "", ResolvedModule{}, fmt.Errorf("resolve package target %q module ownership: %w", importPath, err)
		}
		if !same {
			return "", ResolvedModule{}, fmt.Errorf("package target %q crosses a nested module boundary", importPath)
		}
		return dir, module, nil
	}
	return "", ResolvedModule{}, fmt.Errorf("package target %q is outside the effective module graph", importPath)
}

// retargetEffectiveGraph keeps the caller's selected build list and only
// changes which already-resolved module owns the class project. This is
// essential for package targets: running go list from a dependency directory
// would silently switch to that dependency's standalone graph.
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

func graphEnvironment(base []string, goWork string) []string {
	// Graph policy is passed as argv. Clearing inherited GOFLAGS prevents an
	// ambient unsupported flag from changing the authoritative graph command.
	env := replaceEnv(base, "GOFLAGS", "")
	if goWork != "" {
		env = replaceEnv(env, "GOWORK", goWork)
	}
	return env
}

func replaceEnv(env []string, key, value string) []string {
	prefix := key + "="
	ret := make([]string, 0, len(env)+1)
	for _, item := range env {
		if !strings.HasPrefix(item, prefix) {
			ret = append(ret, item)
		}
	}
	return append(ret, prefix+value)
}

func canonicalExistingDir(path string) (string, error) {
	canonical, err := canonicalExistingPath(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", path)
	}
	return canonical, nil
}

func canonicalExistingFile(path string) (string, error) {
	canonical, err := canonicalExistingPath(path)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(canonical)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("%q is not a regular non-symlink file", path)
	}
	return canonical, nil
}

func canonicalExistingPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(filepath.Clean(abs))
	if err != nil {
		return "", err
	}
	return canonical, nil
}

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func commandError(name string, err error, stderr string) error {
	if message := strings.TrimSpace(stderr); message != "" {
		return fmt.Errorf("%s: %w: %s", name, err, message)
	}
	return fmt.Errorf("%s: %w", name, err)
}
