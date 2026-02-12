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

// First retrieves the first item from the provided sequence. If the sequence is
// empty, it returns ErrNotFound.
func First[T any, Seq ~func(func(T) bool)](seq Seq) (ret T, err error) {
	err = ErrNotFound
	seq(func(item T) bool {
		ret, err = item, nil
		return false
	})
	return
}

// Single retrieves a single item from the provided sequence. If the sequence is
// empty, it returns ErrNotFound. If the sequence contains more than one item, it
// returns ErrMultiEntities.
func Single[T any, Seq ~func(func(T) bool)](seq Seq) (ret T, err error) {
	err = ErrNotFound
	first := true
	seq(func(item T) bool {
		if first {
			ret, err = item, nil
			first = false
			return true
		}
		err = ErrMultiEntities
		return false
	})
	return
}

// Collect retrieves all items from the provided sequence.
func Collect[T any, Seq ~func(func(T) bool)](seq Seq) []T {
	ret := make([]T, 0, 8)
	seq(func(item T) bool {
		ret = append(ret, item)
		return true
	})
	return ret
}

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
