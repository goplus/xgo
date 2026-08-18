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

func preparePolicies(ctx context.Context, cwd string, cli []string) (parsedFlags, error) {
	goCommand, err := hostGoCommand()
	if err != nil {
		return parsedFlags{}, err
	}
	ambient, err := goEnvValue(ctx, goCommand, cwd, nil, "GOFLAGS")
	if err != nil {
		return parsedFlags{}, err
	}
	policy, err := parseRuntimeFlags(cwd, goCommand, "off", ambient, cli)
	if err != nil {
		return parsedFlags{}, err
	}
	policy = sanitizeGraphFlags(policy)
	// GOWORK does not depend on -mod/-modfile/-overlay. Keep graph flags out of
	// GOFLAGS entirely: their canonical paths are passed as distinct argv
	// elements to every graph command, which also preserves spaces and Windows
	// path separators without a second quoting grammar.
	goWork, err := goEnvValue(ctx, goCommand, cwd, []string{}, "GOWORK")
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
// Their original spelling is retained as a rejected runtime flag, so a
// matched runtime still reports the policy error through BuildPolicy while a
// legacy target can continue with its normal command path.
func sanitizeGraphFlags(policy parsedFlags) parsedFlags {
	flags := make([]string, 0, len(policy.graph.Flags))
	for _, flag := range policy.graph.Flags {
		name := ""
		if strings.HasPrefix(flag, "-modfile=") {
			name = "modfile"
		} else if strings.HasPrefix(flag, "-overlay=") {
			name = "overlay"
		}
		if name != "" {
			path := strings.TrimPrefix(flag, "-"+name+"=")
			if _, err := os.Stat(path); err != nil {
				policy.rejected = append(policy.rejected, flag)
				continue
			}
		}
		flags = append(flags, flag)
	}
	policy.graph.Flags = flags
	return policy
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

func goEnvValue(ctx context.Context, goCommand, dir string, graphFlags []string, key string) (string, error) {
	args := append([]string{"env"}, key)
	cmd := exec.CommandContext(ctx, goCommand, args...)
	cmd.Dir = dir
	if graphFlags == nil {
		cmd.Env = os.Environ()
	} else {
		cmd.Env = graphEnvironment(os.Environ(), "", graphFlags)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", commandError("go env "+key, err, stderr.String())
	}
	return strings.TrimSpace(string(out)), nil
}

func loadEffectiveGraph(ctx context.Context, projectDir string, policy GraphPolicy) (*effectiveGraph, error) {
	args := []string{"list", "-m", "-json"}
	args = append(args, policy.Flags...)
	args = append(args, "all")
	cmd := exec.CommandContext(ctx, policy.GoCommand, args...)
	cmd.Dir = projectDir
	cmd.Env = graphEnvironment(os.Environ(), policy.GoWork, nil)
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
	projectDir, err := canonicalExistingDir(projectDir)
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
	if alternate := graphFlagValue(policy.Flags, "modfile"); alternate != "" {
		modfilePath = alternate
	}
	identity, classPaths, err := readTargetModFile(modfilePath)
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
	cmd.Env = graphEnvironment(os.Environ(), policy.GoWork, nil)
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
	canonical, err := canonicalExistingFile(path)
	if err != nil {
		return fileIdentity{}, nil, fmt.Errorf("effective target modfile: %w", err)
	}
	data, err := os.ReadFile(canonical)
	if err != nil {
		return fileIdentity{}, nil, err
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

func resolvePackageDirectory(graph *effectiveGraph, importPath string) (string, ResolvedModule, error) {
	if importPath == "" || strings.Contains(importPath, "@") || strings.Contains(importPath, "...") {
		return "", ResolvedModule{}, fmt.Errorf("runtime provider does not support package target %q", importPath)
	}
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
		suffix := strings.TrimPrefix(importPath, modulePath)
		dir := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(suffix, "/")))
		canonical, err := canonicalExistingDir(dir)
		if err != nil {
			return "", ResolvedModule{}, fmt.Errorf("package target %q: %w", importPath, err)
		}
		if !pathWithin(root, canonical) {
			return "", ResolvedModule{}, fmt.Errorf("package target %q escapes module %q", importPath, modulePath)
		}
		return canonical, module, nil
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
	identity, classPaths, err := readTargetModFile(modfilePath)
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
	}, nil
}

func moduleContainsPackage(modulePath, packagePath string) bool {
	return packagePath == modulePath || strings.HasPrefix(packagePath, modulePath+"/")
}

func graphFlagValue(flags []string, name string) string {
	prefix := "-" + name + "="
	for i := len(flags) - 1; i >= 0; i-- {
		if strings.HasPrefix(flags[i], prefix) {
			return strings.TrimPrefix(flags[i], prefix)
		}
	}
	return ""
}

func graphEnvironment(base []string, goWork string, graphFlags []string) []string {
	env := replaceEnv(base, "GOFLAGS", joinQuotedFields(graphFlags))
	if goWork != "" {
		env = replaceEnv(env, "GOWORK", goWork)
	}
	return env
}

func joinQuotedFields(fields []string) string {
	var joined strings.Builder
	for i, field := range fields {
		if i != 0 {
			joined.WriteByte(' ')
		}
		joined.WriteByte('"')
		for j := 0; j < len(field); j++ {
			if field[j] == '\\' || field[j] == '"' {
				joined.WriteByte('\\')
			}
			joined.WriteByte(field[j])
		}
		joined.WriteByte('"')
	}
	return joined.String()
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
