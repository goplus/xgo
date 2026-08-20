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
	"os"
	"path/filepath"
	"strings"

	"github.com/goplus/mod/modfile"
)

// resolveOverlayLocalTarget classifies overlay-only local targets.
// xgoprojs may parse both files and directories as DirProj.
func (r *Resolver) resolveOverlayLocalTarget(ctx context.Context, candidate string, recursive bool) (TargetKind, string, string, *effectiveGraph, error) {
	graph, err := loadEffectiveGraph(ctx, r.cwd, r.policy.graph)
	if err != nil {
		if errors.Is(err, errNoGoModule) {
			return 0, "", "", nil, ErrNotHandled
		}
		return 0, "", "", nil, err
	}
	logical := overlayPath(r.cwd, candidate)
	projectDir, dirErr := graph.files.canonicalDir(logical)
	kind := TargetDirectory
	expectedFile := ""
	if dirErr != nil {
		if recursive {
			if os.IsNotExist(dirErr) {
				return 0, "", "", nil, ErrNotHandled
			}
			return 0, "", "", nil, fmt.Errorf("overlay local target %q: %w", candidate, dirErr)
		}
		visible, fileErr := graph.files.regularFileVisible(logical)
		if fileErr != nil {
			return 0, "", "", nil, fmt.Errorf("overlay local target %q: %w", candidate, fileErr)
		}
		if !visible {
			if !os.IsNotExist(dirErr) {
				return 0, "", "", nil, fmt.Errorf("overlay local target %q: %w", candidate, dirErr)
			}
			return 0, "", "", nil, ErrNotHandled
		}
		kind = TargetFile
		expectedFile = logical
		projectDir, err = graph.files.canonicalDir(filepath.Dir(logical))
		if err != nil {
			return 0, "", "", nil, fmt.Errorf("overlay local file target %q: %w", candidate, err)
		}
	}
	targetModule, err := graphModuleContainingDirectory(graph, projectDir)
	if err != nil {
		return 0, "", "", nil, err
	}
	graph, err = retargetEffectiveGraph(graph, targetModule)
	if err != nil {
		return 0, "", "", nil, err
	}
	return kind, projectDir, expectedFile, graph, nil
}

func graphModuleContainingDirectory(graph *effectiveGraph, dir string) (ResolvedModule, error) {
	var match ResolvedModule
	bestRoot := ""
	for _, module := range graph.Modules {
		root := module.Effective().Dir
		if root != "" && pathWithin(root, dir) && len(root) > len(bestRoot) {
			match = module
			bestRoot = root
		}
	}
	if bestRoot == "" {
		return ResolvedModule{}, fmt.Errorf("overlay target directory %q is outside the effective module graph", dir)
	}
	return match, nil
}

func classModuleMarked(classModules []ResolvedModule, modulePath string) bool {
	for _, module := range classModules {
		if module.Selected.Path == modulePath {
			return true
		}
	}
	return false
}

// overlayDriverProjectMatch classifies against overlay metadata only.
// Driver v1 rejects a positive overlay match before execution.
func overlayDriverProjectMatch(projectDir string, graph *effectiveGraph, recursive bool) (bool, error) {
	if graph == nil || graph.files == nil || !graph.files.hasOverlay() {
		return false, fmt.Errorf("overlay classification requires an effective graph file view")
	}
	projects, err := overlayDriverProjects(graph)
	if err != nil {
		return false, err
	}
	if len(projects) == 0 {
		return false, nil
	}
	if recursive {
		return overlayWalkDriverProjects(projectDir, projects, graph.files)
	}
	names, err := graph.files.regularFileNames(projectDir)
	if err != nil {
		return false, err
	}
	for _, name := range names {
		if driverProjectMatches(projects, name) {
			return true, nil
		}
	}
	return false, nil
}

func overlayDriverProjects(graph *effectiveGraph) ([]*modfile.Project, error) {
	target := graph.Target.Effective()
	loaded, err := loadDriverModuleView(graph.TargetModFile.Path, filepath.Join(target.Dir, "gox.mod"), graph.files)
	if err != nil {
		return nil, fmt.Errorf("load overlaid target metadata: %w", err)
	}
	projects := append([]*modfile.Project(nil), loaded.Projects()...)
	for _, class := range graph.ClassModules {
		effective := class.Effective()
		classModule, loadErr := loadDriverModuleView(effective.GoMod, filepath.Join(effective.Dir, "gox.mod"), graph.files)
		if loadErr != nil {
			return nil, fmt.Errorf("load overlaid class module %q metadata: %w", class.Selected.Path, loadErr)
		}
		projects = append(projects, classModule.Projects()...)
	}
	return projects, nil
}

func overlayWalkDriverProjects(root string, projects []*modfile.Project, view *graphFileView) (bool, error) {
	root = overlayPath(view.workDir, root)
	visited := make(map[string]struct{})
	var walk func(string) error
	walk = func(current string) error {
		if _, ok := visited[current]; ok {
			return nil
		}
		visited[current] = struct{}{}
		names, err := view.regularFileNames(current)
		if err != nil {
			return err
		}
		for _, name := range names {
			if driverProjectMatches(projects, name) {
				return errDriverProjectInPattern
			}
		}
		dirs, err := view.directoryNames(current)
		if err != nil {
			return err
		}
		for _, name := range dirs {
			if name == "vendor" || name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				continue
			}
			child := filepath.Join(current, name)
			if isGoMod, err := view.regularFileVisible(filepath.Join(child, "go.mod")); err != nil {
				return err
			} else if isGoMod {
				continue
			}
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	err := walk(root)
	if errors.Is(err, errDriverProjectInPattern) {
		return true, nil
	}
	return false, err
}

func unsupportedOverlayError(path string) error {
	return fmt.Errorf("driver v1 does not support flag -overlay=%s", path)
}
