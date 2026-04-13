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

package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/qiniu/x/errors"
)

// configFormat describes a supported configuration file format.
type configFormat struct {
	indexFile string // source filename, e.g. "index.json"
	packFile  string // packed output filename, e.g. "index_pack.json"
}

const (
	indexJSON = iota
	indexYML
	indexYAML
	indexFormatMax
)

var configFormats = [...]configFormat{
	indexJSON: {"index.json", "index_pack.json"},
	indexYML:  {"index.yml", "index_pack.yml"},
	indexYAML: {"index.yaml", "index_pack.yaml"},
}

// -----------------------------------------------------------------------------

// PackFlags controls the behavior of Pack.
type PackFlags int

const (
	// PackFlagTest enables test mode: verify that all index_pack.* files
	// exist and match what Pack would produce, without writing any files.
	PackFlagTest PackFlags = 1 << iota
	PackFlagPrompt
)

// Pack discovers pack roots in the directory tree rooted at dir, merges
// child configuration files into each root, and writes the packed output.
//
// In test mode (PackFlagTest), no files are written; instead Pack verifies
// that every index_pack.* file already exists and matches the expected content.
func Pack(dir string, flags PackFlags) error {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("pack: %w", err)
	}

	projects, err := discoverProjects(dir)
	if err != nil {
		return err
	}

	var errs errors.List
	for _, proj := range projects {
		if err := processPack(proj, flags, dir); err != nil {
			errs.Add(err)
		}
	}
	return errs.ToError()
}

// -----------------------------------------------------------------------------

// PackProject merges all indexFile configuration files found under dir into a
// single packed document and returns its serialised content.
//
// fsys is the filesystem to read from (may be a ZIP-backed fs.ReadDirFS).
// dir is the root directory within fsys that contains the root configuration file.
// indexFile is the filename of the root configuration file (e.g. "index.json").
//
// The returned []byte is the fully-merged configuration in the same format as
// indexFile (JSON, YAML, or YAML with .yml extension). The caller is responsible
// for writing or caching the result; PackProject never writes to any filesystem.
func PackProject(
	fsys fs.ReadDirFS,
	dir string,
	indexFile string,
) (indexPackContent []byte, err error) {
	// Read and parse the root configuration file.
	rootPath := joinFSPath(dir, indexFile)
	isJSON := strings.HasSuffix(indexFile, ".json")
	if !isJSON && !strings.HasSuffix(indexFile, ".yml") && !strings.HasSuffix(indexFile, ".yaml") {
		return nil, fmt.Errorf("pack: unsupported index file format: %s", indexFile)
	}
	rootObj, err := readConfigFS(fsys, rootPath, isJSON, false)
	if err != nil {
		return nil, err
	}

	// Recursively discover and merge child configuration files.
	if err := mergeChildren(fsys, dir, indexFile, rootObj, isJSON); err != nil {
		return nil, err
	}

	packed, err := marshalConfig(rootObj, isJSON)
	if err != nil {
		return nil, fmt.Errorf("pack: marshaling output: %w", err)
	}
	return packed, nil
}

func mergeChildren(fsys fs.ReadDirFS, current, indexFile string, obj map[string]any, isJSON bool) error {
	entries, err := fsys.ReadDir(current)
	if err != nil {
		return fmt.Errorf("pack: reading directory %s: %w", current, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		fname := entry.Name()
		childDir := joinFSPath(current, fname)
		childFile := joinFSPath(childDir, indexFile)
		childObj, err := readConfigFS(fsys, childFile, isJSON, true)
		if err != nil {
			return err
		}
		if err := mergeChildren(fsys, childDir, indexFile, childObj, isJSON); err != nil {
			return err
		}
		if len(childObj) > 0 {
			if _, ok := obj[fname]; ok {
				return fmt.Errorf(
					"pack: collision: key %q already exists at path %q",
					fname, childDir,
				)
			}
			obj[fname] = childObj
		}
	}
	return nil
}

// -----------------------------------------------------------------------------

// projectEntry represents a project.
type projectEntry struct {
	dir    string // absolute directory path
	format int    // indexJSON, indexYML, or indexYAML
}

func discoverProjects(root string) ([]projectEntry, error) {
	var projs []projectEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		format := detectConfigIn(path)
		if format < 0 {
			return nil
		}
		projs = append(projs, projectEntry{dir: path, format: format})
		return fs.SkipDir
	})
	if err != nil {
		return nil, fmt.Errorf("pack: walking directory tree: %w", err)
	}
	return projs, nil
}

func detectConfigIn(dir string) int {
	for format := range indexFormatMax {
		path := filepath.Join(dir, configFormats[format].indexFile)
		if _, err := os.Stat(path); err == nil {
			return format
		}
	}
	return -1
}

// relPath returns path relative to root for use in error messages.
// Falls back to the absolute path if the relative path cannot be computed.
func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

// -----------------------------------------------------------------------------

// processPack merges children into a pack root and writes (or verifies)
// the packed output file.
func processPack(proj projectEntry, flags PackFlags, root string) error {
	if (flags & PackFlagPrompt) != 0 {
		fmt.Fprintln(os.Stderr, "Pack", relPath(root, proj.dir), "...")
	}

	fsys := os.DirFS(proj.dir).(fs.ReadDirFS)
	indexFile := configFormats[proj.format].indexFile
	packed, err := PackProject(fsys, ".", indexFile)
	if err != nil {
		return err
	}

	packFile := filepath.Join(proj.dir, configFormats[proj.format].packFile)
	if flags&PackFlagTest != 0 {
		return verifyPackFile(packFile, root, packed)
	}
	return os.WriteFile(packFile, packed, 0644)
}

// -----------------------------------------------------------------------------

func marshalConfig(obj map[string]any, isJSON bool) ([]byte, error) {
	if isJSON {
		return json.MarshalIndent(obj, "", "\t")
	}
	return yaml.Marshal(obj)
}

// verifyPackFile checks that the file at path exists and its content matches
// expected exactly.
func verifyPackFile(path, root string, expected []byte) error {
	existing, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("pack -t: missing: %s", relPath(root, path))
		}
		return fmt.Errorf("pack -t: reading %s: %w", relPath(root, path), err)
	}
	if !bytes.Equal(existing, expected) {
		return fmt.Errorf("pack -t: out of date: %s", relPath(root, path))
	}
	return nil
}

// -----------------------------------------------------------------------------

// readConfigFS reads and parses a configuration file from an fs.FS.
func readConfigFS(fsys fs.FS, filePath string, isJSON, allowNotExist bool) (map[string]any, error) {
	data, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		if allowNotExist && errors.Is(err, fs.ErrNotExist) {
			return make(map[string]any), nil
		}
		return nil, fmt.Errorf("pack: reading %s: %w", filePath, err)
	}
	var obj map[string]any
	if isJSON {
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, fmt.Errorf("pack: parsing %s: %w", filePath, err)
		}
	} else {
		if err := yaml.Unmarshal(data, &obj); err != nil {
			return nil, fmt.Errorf("pack: parsing %s: %w", filePath, err)
		}
	}
	if obj == nil {
		obj = make(map[string]any)
	}
	return obj, nil
}

// joinFSPath joins a directory and filename using forward slashes,
// as required by fs.FS path conventions.
func joinFSPath(dir, name string) string {
	if dir == "." {
		return name
	}
	return dir + "/" + name
}

// -----------------------------------------------------------------------------
