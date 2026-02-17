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

package fs

import (
	"errors"
	"io/fs"
	"iter"
	"os"
	"path"
	"time"

	"github.com/goplus/xgo/dql"
)

// -----------------------------------------------------------------------------

// Node represents a file or directory in the file system.
type Node struct {
	// Path is the absolute path to the file or directory (relative to the root
	// of the file system).
	Path string

	// directory entry for the file or directory.
	de  fs.DirEntry
	fi  fs.FileInfo
	err error
}

// Name returns the name of the file (or subdirectory) described by the entry.
// This name is only the final element of the path (the base name), not the entire path.
// For example, Name would return "hello.go" not "home/gopher/hello.go".
func (p *Node) Name() (string, error) {
	if p.err != nil {
		return "", p.err
	}
	return p.de.Name(), nil
}

// IsDir reports whether the entry describes a directory.
func (p *Node) IsDir() (bool, error) {
	if p.err != nil {
		return false, p.err
	}
	return p.de.IsDir(), nil
}

// Size returns the size of the file in bytes.
// If the file is a directory, the size is system-dependent and should not be used.
func (p *Node) Size() (int64, error) {
	if p.fi == nil {
		if p.err == nil {
			p.fi, p.err = p.de.Info()
		}
		if p.err != nil {
			return 0, p.err
		}
	}
	return p.fi.Size(), nil
}

// Type returns the type bits for the entry.
// The type bits are a subset of the usual FileMode bits, those returned by the FileMode.Type method.
func (p *Node) Type() (fs.FileMode, error) {
	if p.err != nil {
		return 0, p.err
	}
	return p.de.Type(), nil
}

// Mode returns file mode bits.
func (p *Node) Mode() (fs.FileMode, error) {
	if p.fi == nil {
		if p.err == nil {
			p.fi, p.err = p.de.Info()
		}
		if p.err != nil {
			return 0, p.err
		}
	}
	return p.fi.Mode(), nil
}

// ModTime returns the modification time of the file.
func (p *Node) ModTime() (time.Time, error) {
	if p.fi == nil {
		if p.err == nil {
			p.fi, p.err = p.de.Info()
		}
		if p.err != nil {
			return time.Time{}, p.err
		}
	}
	return p.fi.ModTime(), nil
}

// Sys returns the underlying data source (can return nil).
func (p *Node) Sys() (any, error) {
	if p.fi == nil {
		if p.err == nil {
			p.fi, p.err = p.de.Info()
		}
		if p.err != nil {
			return nil, p.err
		}
	}
	return p.fi.Sys(), nil
}

// -----------------------------------------------------------------------------

// NodeSet represents a set of file system nodes, along with any error that
// occurred while retrieving them.
type NodeSet struct {
	Data iter.Seq[*Node]
	Base fs.FS
	Err  error
}

// Root creates a NodeSet containing the provided root node.
func Root(root fs.FS, doc *Node) NodeSet {
	return NodeSet{
		Base: root,
		Data: func(yield func(*Node) bool) {
			yield(doc)
		},
	}
}

// Nodes creates a NodeSet containing the provided nodes.
func Nodes(root fs.FS, nodes ...*Node) NodeSet {
	return NodeSet{
		Base: root,
		Data: func(yield func(*Node) bool) {
			for _, node := range nodes {
				if !yield(node) {
					break
				}
			}
		},
	}
}

// Dir returns a NodeSet for the specified directory.
func Dir(dir string) NodeSet {
	return New(os.DirFS(dir))
}

// New creates a NodeSet for the provided file system, starting with
// the root node.
func New(root fs.FS) NodeSet {
	return NodeSet{
		Base: root,
		Data: func(yield func(*Node) bool) {
			yield(rootEntry)
		},
	}
}

var (
	rootEntry = &Node{
		Path: "",
		de:   rootDirEntry{},
		fi:   rootDirEntry{},
	}
)

type rootDirEntry struct {
}

func (p rootDirEntry) Name() string {
	return ""
}

func (p rootDirEntry) Size() int64 {
	return 0
}

func (p rootDirEntry) Type() fs.FileMode {
	return fs.ModeDir
}

func (p rootDirEntry) Mode() fs.FileMode {
	return fs.ModeDir
}

func (p rootDirEntry) Info() (fs.FileInfo, error) {
	return p, nil
}

func (p rootDirEntry) ModTime() time.Time {
	return time.Now()
}

func (p rootDirEntry) IsDir() bool {
	return true
}

func (p rootDirEntry) Sys() any {
	return nil
}

// -----------------------------------------------------------------------------

// XGo_Enum returns an iterator over the nodes in the NodeSet.
func (p NodeSet) XGo_Enum() iter.Seq[NodeSet] {
	if p.Err != nil {
		return dql.NopIter[NodeSet]
	}
	return func(yield func(NodeSet) bool) {
		p.Data(func(node *Node) bool {
			return yield(Root(p.Base, node))
		})
	}
}

// Dir returns a NodeSet containing all child nodes of the nodes in the NodeSet
// that are directories.
func (p NodeSet) Dir() NodeSet {
	if p.Err != nil {
		return NodeSet{Err: p.Err}
	}
	return NodeSet{
		Base: p.Base,
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				return yieldChildNodes(p.Base, node, filterDir, yield)
			})
		},
	}
}

// File returns a NodeSet containing all child nodes of the nodes in the NodeSet
// that are files (not directories).
func (p NodeSet) File() NodeSet {
	if p.Err != nil {
		return NodeSet{Err: p.Err}
	}
	return NodeSet{
		Base: p.Base,
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				return yieldChildNodes(p.Base, node, filterFile, yield)
			})
		},
	}
}

// XGo_Child returns a NodeSet containing all child nodes of the nodes in the NodeSet.
func (p NodeSet) XGo_Child() NodeSet {
	if p.Err != nil {
		return NodeSet{Err: p.Err}
	}
	return NodeSet{
		Base: p.Base,
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				return yieldChildNodes(p.Base, node, nil, yield)
			})
		},
	}
}

type filterType = func(fs.DirEntry) bool

func filterDir(de fs.DirEntry) bool {
	return de.IsDir()
}

func filterFile(de fs.DirEntry) bool {
	return !de.IsDir()
}

// yieldChildNodes yields all child nodes of the given node.
func yieldChildNodes(base fs.FS, node *Node, filter filterType, yield func(*Node) bool) bool {
	var items []fs.DirEntry
	var path = node.Path
	isDir, err := node.IsDir()
	if err == nil {
		if !isDir {
			return true
		}
		dir := path
		if dir == "" {
			// fs.ReadDir does not accept an empty string as the directory, use "."
			// instead to read the root directory.
			dir = "."
		}
		items, err = fs.ReadDir(base, dir)
	}
	if err != nil {
		return yield(&Node{Path: path, err: err}) // yield the error as a node
	}
	if path != "" {
		path += "/"
	}
	for _, item := range items {
		if filter == nil || filter(item) {
			childPath := path + item.Name()
			if !yield(&Node{Path: childPath, de: item}) {
				return false
			}
		}
	}
	return true
}

const (
	kindAny = iota
	kindFile
	kindDir
)

// XGo_Any returns a NodeSet containing all descendant nodes (including the
// nodes themselves) with the specified name.
// If name is "file", it returns all file nodes.
// If name is "dir", it returns all directory nodes.
// If name is "", it returns all nodes.
//   - .**.file
//   - .**.dir
//   - .**.*
func (p NodeSet) XGo_Any(name string) NodeSet {
	if p.Err != nil {
		return p
	}
	kind := kindAny
	switch name {
	case "file":
		kind = kindFile
	case "dir":
		kind = kindDir
	case "":
	default:
		return NodeSet{Err: errors.New("XGo_Any: invalid name - " + name)}
	}
	return NodeSet{
		Base: p.Base,
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				return yieldAnyNodes(kind, p.Base, node, yield)
			})
		},
	}
}

// yieldAnyNodes yields all descendant nodes of the given node that match the
// specified kind. If kind is kindAny, it yields all nodes.
func yieldAnyNodes(kind int, base fs.FS, node *Node, yield func(*Node) bool) bool {
	isDir, err := node.IsDir()
	if err != nil {
		return yield(&Node{Path: node.Path, err: err}) // yield the error as a node
	}
	var needDir bool
	var filter filterType
	switch kind {
	case kindFile:
		filter = filterFile
	case kindDir:
		filter = filterDir
		needDir = true
	}
	if kind == kindAny || isDir == needDir {
		if !yield(node) {
			return false
		}
	}
	if isDir {
		return yieldChildNodes(base, node, filter, func(n *Node) bool {
			return yieldAnyNodes(kind, base, n, yield)
		})
	}
	return true
}

// -----------------------------------------------------------------------------

// Match returns a NodeSet containing all child nodes of the nodes in the NodeSet
// that match the specified pattern. The pattern syntax is the same as in path.Match.
// For example, "file*.txt" matches "file1.txt" and "file2.txt", but not "myfile.txt".
func (p NodeSet) Match(pattern string) NodeSet {
	if p.Err != nil {
		return NodeSet{Err: p.Err}
	}
	return NodeSet{
		Base: p.Base,
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				if name, err := node.Name(); err == nil {
					matched, _ := path.Match(pattern, name)
					if matched {
						return yield(node)
					}
				}
				return true
			})
		},
	}
}

// OnError calls onErr for any error in the NodeSet and returns a new NodeSet without
// the nodes that have errors. If onErr returns false, it stops processing and returns
// a NodeSet without the remaining nodes.
func (p NodeSet) OnError(onErr func(error) bool) NodeSet {
	if p.Err != nil {
		onErr(p.Err)
		return NodeSet{Err: p.Err}
	}
	return NodeSet{
		Base: p.Base,
		Data: func(yield func(*Node) bool) {
			p.Data(func(node *Node) bool {
				if node.err != nil {
					return onErr(node.err)
				}
				return yield(node)
			})
		},
	}
}

// -----------------------------------------------------------------------------

// All returns a NodeSet containing all nodes.
// It's a cache operation for performance optimization when you need to traverse
// the nodes multiple times.
func (p NodeSet) All() NodeSet {
	if p.Err != nil {
		return NodeSet{Err: p.Err}
	}
	nodes := dql.Collect(p.Data)
	return Nodes(p.Base, nodes...)
}

// One returns a NodeSet containing the first node.
// It's a performance optimization when you only need the first node (stop early).
func (p NodeSet) One() NodeSet {
	if p.Err != nil {
		return NodeSet{Err: p.Err}
	}
	n, err := dql.First(p.Data)
	if err != nil {
		return NodeSet{Err: err}
	}
	return Root(p.Base, n)
}

// Single returns a NodeSet containing the single node.
// If there are zero or more than one nodes, it returns an error.
// ErrNotFound or ErrMultiEntities is returned accordingly.
func (p NodeSet) Single() NodeSet {
	if p.Err != nil {
		return NodeSet{Err: p.Err}
	}
	n, err := dql.Single(p.Data)
	if err != nil {
		return NodeSet{Err: err}
	}
	return Root(p.Base, n)
}

// -----------------------------------------------------------------------------

// _first returns the first node in the NodeSet.
// It's required by XGo compiler.
func (p NodeSet) XGo_first() (ret *Node, err error) {
	if p.Err != nil {
		err = p.Err
		return
	}
	return dql.First(p.Data)
}

// First returns the first node in the NodeSet.
func (p NodeSet) First() (*Node, error) {
	if p.Err != nil {
		return nil, p.Err
	}
	return dql.First(p.Data)
}

// Collect retrieves all nodes from the NodeSet.
func (p NodeSet) Collect() ([]*Node, error) {
	if p.Err != nil {
		return nil, p.Err
	}
	return dql.Collect(p.Data), nil
}

// -----------------------------------------------------------------------------

// Path returns the path of the first node in the NodeSet.
// The path is the absolute path to the file or directory (relative to the root
// of the file system).
// Note the path is not started with a slash.
// For example, if the root of the file system is "/home/gopher" and the node
// represents the file "/home/gopher/a/b.go", Path would return "a/b.go".
// For the root node, Path would return "" (not "/").
func (p NodeSet) Path() (name string, err error) {
	node, err := p.First()
	if err != nil {
		return
	}
	return node.Path, nil
}

// Name returns the name of the first node in the NodeSet.
func (p NodeSet) Name() (name string, err error) {
	node, err := p.First()
	if err != nil {
		return
	}
	return node.Name()
}

// IsDir reports whether the first node in the NodeSet is a directory.
func (p NodeSet) IsDir() (is bool, err error) {
	node, err := p.First()
	if err != nil {
		return
	}
	return node.IsDir()
}

// Size returns the size of the first node in the NodeSet.
func (p NodeSet) Size() (size int64, err error) {
	node, err := p.First()
	if err != nil {
		return
	}
	return node.Size()
}

// Mode returns the file mode of the first node in the NodeSet.
func (p NodeSet) Mode() (mode fs.FileMode, err error) {
	node, err := p.First()
	if err != nil {
		return
	}
	return node.Mode()
}

// ModTime returns the modification time of the first node in the NodeSet.
func (p NodeSet) ModTime() (modTime time.Time, err error) {
	node, err := p.First()
	if err != nil {
		return
	}
	return node.ModTime()
}

// -----------------------------------------------------------------------------
