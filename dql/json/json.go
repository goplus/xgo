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

package json

import (
	"bytes"
	"encoding/json"
	"io"
	"iter"

	"github.com/goplus/xgo/dql/maps"
	"github.com/qiniu/x/stream"
)

const (
	XGoPackage = "github.com/goplus/xgo/dql/maps"
)

// -----------------------------------------------------------------------------

// Node represents a map[string]any node.
type Node = maps.Node

// NodeSet represents a set of JSON nodes.
type NodeSet = maps.NodeSet

// New creates a JSON NodeSet from JSON data read from r.
func New(r io.Reader) NodeSet {
	var data map[string]any
	err := json.NewDecoder(r).Decode(&data)
	if err != nil {
		return NodeSet{Err: err}
	}
	return maps.New(data)
}

// Source creates a JSON NodeSet from various source types:
// - string: treats the string as a file path, opens the file, and reads JSON data from it.
// - []byte: reads JSON data from the byte slice.
// - io.Reader: reads JSON data from the provided reader.
// - map[string]any: creates a NodeSet from the provided map.
// - Node: creates a NodeSet containing the single provided node.
// - iter.Seq[Node]: directly uses the provided sequence of nodes.
// - NodeSet: returns the provided NodeSet as is.
// If the source type is unsupported, it panics.
func Source(r any) (ret NodeSet) {
	switch v := r.(type) {
	case string:
		f, err := stream.Open(v)
		if err != nil {
			return NodeSet{Err: err}
		}
		defer f.Close()
		return New(f)
	case []byte:
		r := bytes.NewReader(v)
		return New(r)
	case io.Reader:
		return New(v)
	case map[string]any:
		return maps.New(v)
	case Node:
		return maps.Root(v)
	case iter.Seq[Node]:
		return NodeSet{Data: v}
	case NodeSet:
		return v
	default:
		panic("dql/json.Source: unsupport source type")
	}
}

// -----------------------------------------------------------------------------
