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
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/qiniu/x/errors"
)

// PackFlags controls the behavior of Pack.
type PackFlags int

const (
	// PackFlagTest enables test mode: verify that all index_pack.* files
	// exist and match what Pack would produce, without writing any files.
	PackFlagTest PackFlags = 1 << iota
	PackFlagPrompt
)

// configFormat describes a supported configuration file format.
type configFormat struct {
	source string // source filename, e.g. "index.json"
	packed string // packed output filename, e.g. "index_pack.json"
	ext    string // file extension, e.g. ".json"
}

const (
	indexJSON = iota
	indexYML
	indexYAML
	indexFormatMax
)

var configFormats = [...]configFormat{
	indexJSON: {"index.json", "index_pack.json", ".json"},
	indexYML:  {"index.yml", "index_pack.yml", ".yml"},
	indexYAML: {"index.yaml", "index_pack.yaml", ".yaml"},
}

// configEntry records a directory that contains a configuration file.
type configEntry struct {
	dir    string // absolute directory path
	format int    // indexJSON, indexYML, or indexYAML
}

// packGroup represents a pack root and its child configuration entries.
type packGroup struct {
	root     configEntry
	children []configEntry
}

// -----------------------------------------------------------------------------

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

	configs, err := discoverConfigs(dir)
	if err != nil {
		return err
	}

	if len(configs) == 0 {
		fmt.Fprintf(os.Stderr, "warning: no index.* files found in %s\n", dir)
		return nil
	}

	groups, err := groupByPackRoot(configs, dir)
	if err != nil {
		return err
	}

	var errs errors.List
	for _, g := range groups {
		if err := processPack(g, flags, dir); err != nil {
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
	format, err := formatFromFilename(indexFile)
	if err != nil {
		return nil, err
	}

	// Read and parse the root configuration file.
	rootPath := joinFSPath(dir, indexFile)
	rootObj, err := readConfigFS(fsys, rootPath, dir, format)
	if err != nil {
		return nil, err
	}

	// Recursively discover and merge child configuration files.
	if err := mergeChildren(fsys, dir, dir, indexFile, format, rootObj); err != nil {
		return nil, err
	}

	packed, err := marshalConfig(rootObj, format)
	if err != nil {
		return nil, fmt.Errorf("pack: marshaling output: %w", err)
	}
	return packed, nil
}

// mergeChildren recursively reads subdirectories of current within fsys using
// ReadDir, and merges any indexFile found into rootObj at the path relative to
// rootDir.
func mergeChildren(fsys fs.ReadDirFS, current, rootDir, indexFile string, format int, rootObj map[string]any) error {
	entries, err := fsys.ReadDir(current)
	if err != nil {
		return fmt.Errorf("pack: reading directory %s: %w", fsRelPath(rootDir, current), err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		childDir := joinFSPath(current, entry.Name())
		childFile := joinFSPath(childDir, indexFile)
		if _, err := fs.Stat(fsys, childFile); err == nil {
			childObj, err := readConfigFS(fsys, childFile, rootDir, format)
			if err != nil {
				return err
			}
			segments := strings.Split(fsRelPath(rootDir, childDir), "/")
			if err := mergeAtPath(rootObj, segments, childObj, fsRelPath(rootDir, childFile)); err != nil {
				return err
			}
		}
		if err := mergeChildren(fsys, childDir, rootDir, indexFile, format, rootObj); err != nil {
			return err
		}
	}
	return nil
}

// -----------------------------------------------------------------------------

// discoverConfigs walks the directory tree and returns every directory
// that contains a configuration file (index.json, index.yml, or index.yaml).
// It reports a fatal error if any directory contains more than one format.
func discoverConfigs(root string) ([]configEntry, error) {
	var configs []configEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		entry, err := detectConfigIn(path, root)
		if err != nil {
			return err
		}
		if entry != nil {
			configs = append(configs, *entry)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("pack: walking directory tree: %w", err)
	}
	return configs, nil
}

// detectConfigIn checks whether dir contains exactly one of the recognized
// configuration files. Returns nil if none is found; returns an error if
// more than one is found.
func detectConfigIn(dir, root string) (*configEntry, error) {
	var found *configEntry
	for format := range indexFormatMax {
		source := configFormats[format].source
		path := filepath.Join(dir, source)
		if _, err := os.Stat(path); err == nil {
			if found != nil {
				return nil, fmt.Errorf("pack: directory %s contains multiple config files: %s and %s",
					relPath(root, dir), configFormats[found.format].source, source)
			}
			found = &configEntry{dir: dir, format: format}
		}
	}
	return found, nil
}

// groupByPackRoot partitions config entries into pack groups. A config entry
// is a pack root if no ancestor directory (that also has a config file) exists
// in the list. All other entries become children of the nearest ancestor root.
func groupByPackRoot(configs []configEntry, root string) ([]packGroup, error) {
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].dir < configs[j].dir
	})

	var groups []packGroup
	for _, cfg := range configs {
		placed := false
		for i := range groups {
			groupRoot := groups[i].root.dir
			if isSubdirectory(groupRoot, cfg.dir) {
				format := groups[i].root.format
				if cfg.format != format {
					return nil, fmt.Errorf(
						"pack: format mismatch: %s uses %s but pack root %s uses %s",
						relPath(root, filepath.Join(cfg.dir, configFormats[cfg.format].source)), configFormats[cfg.format].ext,
						relPath(root, filepath.Join(groupRoot, configFormats[format].source)), configFormats[format].ext,
					)
				}
				groups[i].children = append(groups[i].children, cfg)
				placed = true
				break
			}
		}
		if !placed {
			groups = append(groups, packGroup{root: cfg})
		}
	}
	return groups, nil
}

// isSubdirectory reports whether child is a proper subdirectory of parent.
func isSubdirectory(parent, child string) bool {
	return strings.HasPrefix(child, parent+string(filepath.Separator))
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
func processPack(g packGroup, flags PackFlags, root string) error {
	if (flags & PackFlagPrompt) != 0 {
		fmt.Fprintln(os.Stderr, "Pack", relPath(root, g.root.dir), "...")
	}

	fsys := os.DirFS(g.root.dir).(fs.ReadDirFS)
	indexFile := configFormats[g.root.format].source
	packed, err := PackProject(fsys, ".", indexFile)
	if err != nil {
		return err
	}

	packFile := filepath.Join(g.root.dir, configFormats[g.root.format].packed)
	if flags&PackFlagTest != 0 {
		return verifyPackFile(packFile, root, packed)
	}
	return os.WriteFile(packFile, packed, 0644)
}

// mergeAtPath nests child into root at the location described by segments.
// Intermediate map nodes are created as needed. A fatal error is returned
// if a key collision is detected.
func mergeAtPath(root map[string]any, segments []string, child map[string]any, childFile string) error {
	current := root
	for i, seg := range segments[:len(segments)-1] {
		if existing, ok := current[seg]; ok {
			if m, ok := existing.(map[string]any); ok {
				current = m
			} else {
				return fmt.Errorf(
					"pack: collision: key %q at path %q is not an object (introduced by directory structure, conflicts with %s)",
					seg, strings.Join(segments[:i+1], "/"), childFile,
				)
			}
		} else {
			m := make(map[string]any)
			current[seg] = m
			current = m
		}
	}

	lastSeg := segments[len(segments)-1]
	if _, exists := current[lastSeg]; exists {
		return fmt.Errorf(
			"pack: collision: key %q already exists at path %q (introduced by %s)",
			lastSeg, strings.Join(segments, "/"), childFile,
		)
	}
	current[lastSeg] = child
	return nil
}

// -----------------------------------------------------------------------------

// marshalConfig serializes obj back to the given format. JSON output uses
// tab indentation and a trailing newline.
func marshalConfig(obj map[string]any, format int) ([]byte, error) {
	switch format {
	case indexJSON:
		data, err := json.MarshalIndent(obj, "", "\t")
		if err != nil {
			return nil, err
		}
		return append(data, '\n'), nil
	case indexYML, indexYAML:
		return yaml.Marshal(obj)
	default:
		return nil, fmt.Errorf("pack: unsupported format: %s", configFormats[format].ext)
	}
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

// formatFromFilename returns the format index for the given config filename.
func formatFromFilename(indexFile string) (int, error) {
	for i := range indexFormatMax {
		if configFormats[i].source == indexFile {
			return i, nil
		}
	}
	return 0, fmt.Errorf("pack: unsupported config file: %s (expected index.json, index.yml, or index.yaml)", indexFile)
}

// readConfigFS reads and parses a configuration file from an fs.FS.
func readConfigFS(fsys fs.FS, filePath, baseDir string, format int) (map[string]any, error) {
	data, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return nil, fmt.Errorf("pack: reading %s: %w", fsRelPath(baseDir, filePath), err)
	}
	var obj map[string]any
	if format == indexJSON {
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, fmt.Errorf("pack: parsing %s: %w", fsRelPath(baseDir, filePath), err)
		}
	} else {
		if err := yaml.Unmarshal(data, &obj); err != nil {
			return nil, fmt.Errorf("pack: parsing %s: %w", fsRelPath(baseDir, filePath), err)
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

// fsRelPath returns target relative to base using forward-slash separators.
func fsRelPath(base, target string) string {
	if base == "." {
		return target
	}
	return strings.TrimPrefix(target, base+"/")
}
