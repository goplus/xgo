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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// graphFileView models -overlay paths for classification only.
// The Go command remains authoritative for graph resolution.
type graphFileView struct {
	workDir            string
	replacements       map[string]string
	replacementParents map[string]struct{}
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
	view.replacementParents = make(map[string]struct{})
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
	for child, target := range view.replacements {
		if target == "" {
			continue
		}
		for parent := filepath.Dir(child); parent != child; parent = filepath.Dir(parent) {
			if parentTarget, ok := view.replacements[parent]; ok && parentTarget != "" {
				return nil, fmt.Errorf("overlay %q maps both file %q and child %q", overlay, parent, child)
			}
			view.replacementParents[parent] = struct{}{}
			if parent == filepath.Dir(parent) {
				break
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

func (v *graphFileView) logicalPath(path string) string {
	if !v.hasOverlay() {
		return path
	}
	return overlayPath(v.workDir, path)
}

func statVisible(path string, matches func(os.FileMode) bool) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return matches(info.Mode()), nil
}

func (v *graphFileView) readFile(path string) ([]byte, error) {
	if !v.hasOverlay() {
		return os.ReadFile(path)
	}
	logical := v.logicalPath(path)
	if actual, ok := v.replacements[logical]; ok {
		if actual == "" {
			return nil, &os.PathError{Op: "read", Path: logical, Err: os.ErrNotExist}
		}
		return os.ReadFile(actual)
	}
	// An exact child replacement wins over an ancestor deletion, as in cmd/go.
	if _, hidden := v.replacementAncestor(logical); hidden {
		return nil, &os.PathError{Op: "read", Path: logical, Err: os.ErrNotExist}
	}
	return os.ReadFile(logical)
}

func (v *graphFileView) replacementAncestor(path string) (string, bool) {
	if !v.hasOverlay() {
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

// Replacement destinations remain opaque until Go reads them.
func (v *graphFileView) hasReplacementChild(path string) bool {
	if !v.hasOverlay() {
		return false
	}
	_, ok := v.replacementParents[v.logicalPath(path)]
	return ok
}

func (v *graphFileView) directoryVisible(dir string) (bool, error) {
	if !v.hasOverlay() {
		return statVisible(dir, os.FileMode.IsDir)
	}
	logical := v.logicalPath(dir)
	if _, exact := v.replacements[logical]; exact {
		return false, nil
	}
	_, hasAncestor := v.replacementAncestor(logical)
	if hasAncestor && !v.hasReplacementChild(logical) {
		return false, nil
	}
	if v.hasReplacementChild(logical) {
		return true, nil
	}
	return statVisible(logical, os.FileMode.IsDir)
}

func (v *graphFileView) physicalEntries(dir string) ([]os.DirEntry, error) {
	if !v.hasOverlay() {
		return os.ReadDir(dir)
	}
	logical := v.logicalPath(dir)
	if _, exact := v.replacements[logical]; exact {
		return nil, nil
	}
	if _, hasAncestor := v.replacementAncestor(logical); hasAncestor {
		return nil, nil
	}
	entries, err := os.ReadDir(logical)
	if os.IsNotExist(err) && v.hasReplacementChild(logical) {
		return nil, nil
	}
	return entries, err
}

func (v *graphFileView) regularFileVisible(path string) (bool, error) {
	if !v.hasOverlay() {
		return statVisible(path, os.FileMode.IsRegular)
	}
	logical := v.logicalPath(path)
	if actual, exact := v.replacements[logical]; exact {
		return actual != "", nil
	}
	if _, hidden := v.replacementAncestor(logical); hidden {
		return false, nil
	}
	return statVisible(logical, os.FileMode.IsRegular)
}

func (v *graphFileView) canonicalFile(path string) (string, error) {
	if !v.hasOverlay() {
		return canonicalExistingFile(path)
	}
	logical := v.logicalPath(path)
	if actual, exact := v.replacements[logical]; exact {
		if actual == "" {
			return "", &os.PathError{Op: "stat", Path: logical, Err: os.ErrNotExist}
		}
		return logical, nil
	}
	if _, hidden := v.replacementAncestor(logical); hidden {
		return "", &os.PathError{Op: "stat", Path: logical, Err: os.ErrNotExist}
	}
	return canonicalExistingFile(logical)
}

// Synthetic directories are safe because overlay-backed drivers never execute.
func (v *graphFileView) canonicalDir(path string) (string, error) {
	if !v.hasOverlay() {
		return canonicalExistingDir(path)
	}
	logical := v.logicalPath(path)
	visible, err := v.directoryVisible(logical)
	if err != nil {
		return "", err
	}
	if !visible {
		return "", &os.PathError{Op: "stat", Path: logical, Err: os.ErrNotExist}
	}
	if v.hasReplacementChild(logical) {
		return logical, nil
	}
	return canonicalExistingDir(logical)
}
