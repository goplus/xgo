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
)

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

// replacementAncestor returns the nearest overlay entry above path.
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

// hasReplacementChild reports whether path has a visible overlay child.
// Replacement destinations remain opaque until Go reads them.
func (v *graphFileView) hasReplacementChild(path string) bool {
	if !v.hasOverlay() {
		return false
	}
	for child, actual := range v.replacements {
		if actual != "" && child != path && pathWithin(path, child) {
			return true
		}
	}
	return false
}

func (v *graphFileView) directoryVisible(dir string) (bool, error) {
	if !v.hasOverlay() {
		return statVisible(dir, os.FileMode.IsDir)
	}
	logical := v.logicalPath(dir)
	if _, exact := v.replacements[logical]; exact {
		return false, nil
	}
	ancestor, hasAncestor := v.replacementAncestor(logical)
	if hasAncestor && ancestor != "" {
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
		if actual == "" {
			return false, nil
		}
		return true, nil
	}
	if _, hidden := v.replacementAncestor(logical); hidden {
		return false, nil
	}
	return statVisible(logical, os.FileMode.IsRegular)
}

// canonicalFile returns a physical path or an overlay-only logical path.
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

// canonicalDir is the directory counterpart to canonicalFile.
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
