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

package html

import (
	"strings"

	"github.com/goplus/xgo/dql/html"
)

const (
	XGoPackage = "github.com/goplus/xgo/dql/html"
)

// Object represents an HTML object.
type Object = *html.Node

// New creates a new HTML object from a string.
func New(text string) (ret Object, err error) {
	return html.Parse(strings.NewReader(text))
}
