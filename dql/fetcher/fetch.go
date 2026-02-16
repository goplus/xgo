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

package fetcher

import (
	"errors"
	"reflect"

	"github.com/goplus/xgo/dql/html"
)

// -----------------------------------------------------------------------------

// Conv defines a converter function type.
// func(input any, doc html.NodeSet) <any-object>
// A converter function converts a html source to an object.
type Conv = any

// convert converts a html source to an object.
func convert(conv reflect.Value, input, source any) any {
	doc := reflect.ValueOf(html.Source(source))
	out := conv.Call([]reflect.Value{reflect.ValueOf(input), doc})
	return out[0].Interface()
}

// -----------------------------------------------------------------------------

var (
	ErrUnknownPageType = errors.New("unknown page type")
)

// Do fetches HTML content from an input and converts it to an object by
// registered converter.
func Do(fetchType string, input any) (any, error) {
	page, ok := convs[fetchType]
	if !ok {
		return nil, ErrUnknownPageType
	}
	url := page.URL(input)
	return convert(page.Conv, input, url), nil
}

// From reads HTML content from a source and converts it to an object by
// registered converter. It is used when HTML content is already available.
func From(fetchType string, input, source any) (any, error) {
	page, ok := convs[fetchType]
	if !ok {
		return nil, ErrUnknownPageType
	}
	return convert(page.Conv, input, source), nil
}

// fetchInfo represents a fetch information, including convert function
// and URL function that generates URL from input.
type fetchInfo struct {
	Conv reflect.Value
	URL  func(input any) string
}

var (
	convs = map[string]fetchInfo{}
)

// Register registers a fetchType with a convert function.
// The urlOf function generates URL from input.
// func conv(input any, doc html.NodeSet) <any-object>
func Register(fetchType string, conv Conv, urlOf func(input any) string) {
	vConv := reflect.ValueOf(conv)
	convs[fetchType] = fetchInfo{vConv, urlOf}
}

// -----------------------------------------------------------------------------
