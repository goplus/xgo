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

package stream

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
)

var (
	// ErrUnknownScheme is returned when an unknown scheme is encountered in a URL.
	ErrUnknownScheme = errors.New("unknown scheme")
)

// -------------------------------------------------------------------------------------

// OpenFunc defines the function type for opening a resource by URL.
type OpenFunc = func(url string) (io.ReadCloser, error)

var (
	openers = map[string]OpenFunc{}
)

// Register registers a scheme with an open function.
func Register(scheme string, open OpenFunc) {
	openers[scheme] = open
}

// Open opens a resource identified by the given URL.
// It supports different schemes by utilizing registered open functions.
// If the URL has no scheme, it is treated as a file path.
func Open(url string) (io.ReadCloser, error) {
	scheme := schemeOf(url)
	if scheme == "" {
		return os.Open(url)
	}
	if open, ok := openers[scheme]; ok {
		return open(url)
	}
	return nil, &fs.PathError{Op: "dql/stream.Open", Err: ErrUnknownScheme, Path: url}
}

func schemeOf(url string) (scheme string) {
	pos := strings.IndexAny(url, ":/")
	if pos > 0 {
		if url[pos] == ':' {
			return url[:pos]
		}
	}
	return ""
}

// -------------------------------------------------------------------------------------
