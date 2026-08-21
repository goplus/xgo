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
