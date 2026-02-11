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

package dql

import (
	"errors"
	"strconv"
	"strings"
)

const (
	XGoPackage = true
)

var (
	ErrNotFound      = errors.New("entity not found")
	ErrMultiEntities = errors.New("too many entities found")
)

// -----------------------------------------------------------------------------

// NopIter is a no-operation iterator that yields no values.
func NopIter[T any](yield func(T) bool) {}

// -----------------------------------------------------------------------------

// Int parses the given string as an integer, removing any commas and trimming
// whitespace.
func Int__0(text string) (int, error) {
	return strconv.Atoi(strings.ReplaceAll(strings.TrimSpace(text), ",", ""))
}

// Int parses the given string as an integer, removing any commas and trimming
// whitespace. If an error occurs, it returns 0 and the error.
func Int__1(text string, err error) (int, error) {
	if err != nil {
		return 0, err
	}
	return Int__0(text)
}

// -----------------------------------------------------------------------------
