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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goplus/mod/modfile"
	"github.com/goplus/mod/modload"
	"github.com/goplus/mod/xgomod"
)

const driverGuardEnv = "XGO_DRIVER_GUARD"

// loadDriverModule preserves metadata fallback and reports unreadable optional files.
func loadDriverModule(goMod, goxMod string) (modload.Module, error) {
	return loadDriverModuleView(goMod, goxMod, nil)
}

func loadDriverModuleView(goMod, goxMod string, view *graphFileView) (modload.Module, error) {
	optionalPaths := map[string]struct{}{goxMod: {}}
	if strings.HasSuffix(goxMod, "gox.mod") {
		optionalPaths[strings.TrimSuffix(goxMod, "gox.mod")+"gop.mod"] = struct{}{}
	}
	var optionalErr error
	loaded, err := modload.LoadFromEx(goMod, goxMod, func(path string) ([]byte, error) {
		data, readErr := view.readFile(path)
		if _, optional := optionalPaths[path]; optional && readErr != nil && !os.IsNotExist(readErr) && optionalErr == nil {
			optionalErr = fmt.Errorf("read optional module metadata %q: %w", path, readErr)
		}
		return data, readErr
	})
	if optionalErr != nil {
		return modload.Module{}, optionalErr
	}
	return loaded, err
}

func externalClassModule(module modload.Module) string {
	for _, require := range module.Require {
		if require.Syntax != nil && modload.HasClassMarker(require.Syntax.Suffix) {
			return require.Mod.Path
		}
	}
	return ""
}

func driverProjectMatches(projects []*modfile.Project, name string) bool {
	ext := modfile.ClassExt(name)
	for _, project := range projects {
		if project != nil && project.Driver != nil && project.Ext == ext && project.IsProj(ext, name) {
			return true
		}
	}
	return false
}

func loadResolvedClasses(graph *effectiveGraph) (*xgomod.Module, bool, error) {
	target := graph.Target.Effective()
	loaded, err := loadDriverModuleView(graph.TargetModFile.Path, filepath.Join(target.Dir, "gox.mod"), graph.files)
	if err != nil {
		return nil, false, err
	}
	hasClass := len(graph.ClassModules) != 0 || loaded.HasProject()
	if !hasClass {
		return nil, false, nil
	}
	module := xgomod.New(loaded)
	if err := module.ImportClassesResolved(toXGoGraph(graph)); err != nil {
		return nil, false, err
	}
	return module, true, nil
}

func toXGoGraph(graph *effectiveGraph) xgomod.ResolvedClassGraph {
	// xgomod needs only the target and marked modules; the full graph remains
	// available for package resolution.
	return xgomod.ResolvedClassGraph{
		Target: graph.Target, ClassModules: graph.ClassModules, TargetModFile: graph.TargetModFile,
	}
}

func findProjectFile(dir string, module *xgomod.Module) (string, *xgomod.ProjectInfo, int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, 0, err
	}
	var (
		projectFile string
		projectInfo *xgomod.ProjectInfo
		count       int
	)
	for _, entry := range entries {
		ext := modfile.ClassExt(entry.Name())
		classInfo, ok := module.LookupClassInfo(ext)
		if !ok || !classInfo.Project.IsProj(ext, entry.Name()) {
			continue
		}
		driverProject := classInfo.Project.Driver != nil
		info, err := entry.Info()
		if err != nil {
			return "", nil, 0, err
		}
		if !info.Mode().IsRegular() {
			if driverProject {
				return "", nil, 0, fmt.Errorf("driver project file %q is not a regular non-symlink file", filepath.Join(dir, entry.Name()))
			}
			continue
		}
		count++
		if driverProject {
			projectFile = filepath.Join(dir, entry.Name())
			projectInfo = classInfo
		}
	}
	return projectFile, projectInfo, count, nil
}

var errDriverProjectInPattern = errors.New("driver-backed project in pattern")

func patternContainsDriverProject(root string, module *xgomod.Module) (bool, error) {
	return walkDriverPattern(root, func(dir string) (bool, error) {
		_, info, _, err := findProjectFile(dir, module)
		return info != nil, err
	})
}

func patternContainsDriverProjects(root string, projects []*modfile.Project) (bool, error) {
	return walkDriverPattern(root, func(dir string) (bool, error) {
		return hasDriverProject(dir, projects)
	})
}

func walkDriverPattern(root string, matches func(string) (bool, error)) (bool, error) {
	err := filepath.WalkDir(root, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		if current != root {
			name := entry.Name()
			if name == "vendor" || name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				return filepath.SkipDir
			}
			goMod := filepath.Join(current, "go.mod")
			if info, err := os.Lstat(goMod); err == nil {
				if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
					return fmt.Errorf("nested module marker %q is not a regular non-symlink file", goMod)
				}
				return filepath.SkipDir
			} else if !os.IsNotExist(err) {
				return err
			}
		}
		matched, err := matches(current)
		if err != nil {
			return err
		}
		if matched {
			return errDriverProjectInPattern
		}
		return nil
	})
	if errors.Is(err, errDriverProjectInPattern) {
		return true, nil
	}
	return false, err
}

func validatePack(projectDir string, pack *modfile.Pack) (string, string, error) {
	if pack == nil {
		return "", "", nil
	}
	dir := filepath.Clean(filepath.FromSlash(pack.Directory))
	if pack.Directory == "" || filepath.IsAbs(dir) || dir == ".." || strings.HasPrefix(dir, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("driver pack directory %q is invalid", pack.Directory)
	}
	if pack.IndexFile == "" || filepath.Base(pack.IndexFile) != pack.IndexFile || strings.ContainsAny(pack.IndexFile, `/\`) {
		return "", "", fmt.Errorf("driver pack index %q is invalid", pack.IndexFile)
	}
	root := filepath.Join(projectDir, dir)
	canonical, err := canonicalExistingDir(root)
	if err != nil {
		return "", "", fmt.Errorf("driver pack directory: %w", err)
	}
	if !pathWithin(projectDir, canonical) {
		return "", "", fmt.Errorf("driver pack directory escapes the project")
	}
	return filepath.ToSlash(dir), pack.IndexFile, nil
}

func sameFile(a, b string) (bool, error) {
	aInfo, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	bInfo, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return os.SameFile(aInfo, bInfo), nil
}

func defaultExecutableName(projectDir, importPath string) string {
	if importPath == "" {
		return filepath.Base(projectDir)
	}
	parts := strings.Split(strings.TrimSuffix(importPath, "/"), "/")
	name := parts[len(parts)-1]
	if majorVersionRE.MatchString(name) && len(parts) > 1 {
		name = parts[len(parts)-2]
	}
	return name
}

var majorVersionRE = regexp.MustCompile(`^v(?:[2-9]|[1-9][0-9]+)$`)

func hasRecursivePattern(target string) bool {
	clean := filepath.ToSlash(filepath.Clean(target))
	return clean == "..." || strings.HasSuffix(clean, "/...")
}

func trimRecursivePattern(target string) string {
	clean := filepath.ToSlash(filepath.Clean(target))
	clean = strings.TrimSuffix(clean, "...")
	clean = strings.TrimSuffix(clean, "/")
	if clean == "" {
		return "."
	}
	return filepath.FromSlash(clean)
}

func sha256Bytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
