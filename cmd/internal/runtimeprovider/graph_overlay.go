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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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

// hasReplacementChild reports whether a non-deleted overlay key is below path.
// It both preserves physical files below deletions and synthesizes virtual
// directories; replacement destinations stay opaque until Go reads them.
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

// regularFileNames merges physical entries with overlay keys, but never stats
// a replacement destination. This keeps ordinary legacy overlays unchanged.
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
	if v.hasReplacementAncestor(logical) {
		return false, nil
	}
	return statVisible(logical, os.FileMode.IsRegular)
}

// canonicalFile pins a physical file in the usual case, but keeps the logical
// path for an overlay-only file. The latter is necessary because cmd/go can
// create go.mod (or an alternate modfile) solely through -overlay.
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
