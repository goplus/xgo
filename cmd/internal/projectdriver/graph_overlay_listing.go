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
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// regularFileNames merges physical and overlay files without inspecting destinations.
func (v *graphFileView) regularFileNames(dir string) ([]string, error) {
	logicalDir := v.logicalPath(dir)
	entries, err := v.physicalEntries(logicalDir)
	if err != nil {
		return nil, err
	}
	files := make(map[string]bool, len(entries))
	for _, entry := range entries {
		logical := filepath.Join(logicalDir, entry.Name())
		if v.hasOverlay() {
			if actual, replaced := v.replacements[logical]; replaced {
				files[entry.Name()] = actual != ""
				continue
			}
			if v.hasReplacementChild(logical) {
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
	if v.hasOverlay() {
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

// directoryNames returns physical children plus overlay-only directories.
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
		if v.hasReplacementChild(logical) {
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
