/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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

package iox

import (
	"bytes"
	"errors"
	"io"
	"os"

	"github.com/qiniu/x/byteutil"
)

// -----------------------------------------------------------------------------

var (
	ErrInvalidSource = errors.New("invalid source")
)

// ReadSource converts src to a []byte if possible; otherwise it returns an error.
// Supported types for src are:
//   - string (as content, NOT as filename)
//   - []byte (as content)
//   - *bytes.Buffer (as content)
//   - io.Reader (as content)
func ReadSource(src any) ([]byte, error) {
	switch s := src.(type) {
	case string:
		return byteutil.Bytes(s), nil
	case []byte:
		return s, nil
	case *bytes.Buffer:
		// is io.Reader, but src is already available in []byte form
		if s != nil {
			return s.Bytes(), nil
		}
	case io.Reader:
		return io.ReadAll(s)
	}
	return nil, ErrInvalidSource
}

// If src != nil, ReadSourceLocal converts src to a []byte if possible;
// otherwise it returns an error. If src == nil, ReadSourceLocal returns
// the result of reading the file specified by filename.
func ReadSourceLocal(filename string, src any) ([]byte, error) {
	if src != nil {
		return ReadSource(src)
	}
	return os.ReadFile(filename)
}

// -----------------------------------------------------------------------------
