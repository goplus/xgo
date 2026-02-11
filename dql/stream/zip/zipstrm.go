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

package zip

import (
	"archive/zip"
	"io"
	"strings"
	"syscall"

	"github.com/goplus/xgo/dql/stream"
)

// -------------------------------------------------------------------------------------

type readCloser struct {
	io.ReadCloser
	zipf *zip.ReadCloser
}

func (p *readCloser) Close() error {
	p.ReadCloser.Close()
	return p.zipf.Close()
}

// Open opens a zipped file object.
func Open(url string) (io.ReadCloser, error) {
	file := strings.TrimPrefix(url, "zip:")
	pos := strings.Index(file, "#")
	if pos <= 0 {
		return nil, syscall.EINVAL
	}
	zipfile, name := file[:pos], file[pos+1:]
	zipf, err := zip.OpenReader(zipfile)
	if err != nil {
		return nil, err
	}
	for _, fi := range zipf.File {
		if fi.Name == name {
			f, err := fi.Open()
			if err != nil {
				return nil, err
			}
			return &readCloser{f, zipf}, nil
		}
	}
	return nil, syscall.ENOENT
}

func init() {
	// zip:file#index.htm
	stream.Register("zip", Open)
}

// -------------------------------------------------------------------------------------
