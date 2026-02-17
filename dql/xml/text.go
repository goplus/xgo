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

package xml

import (
	"encoding/xml"
	"unsafe"

	"github.com/goplus/xgo/dql"
)

// -----------------------------------------------------------------------------

// _text retrieves the text content of the first child xml.CharData.
// It only retrieves from the first node in the NodeSet.
func (p NodeSet) XGo_text__0() string {
	val, _ := p.XGo_text__1()
	return val
}

// _text retrieves the text content of the first child xml.CharData.
// It only retrieves from the first node in the NodeSet.
func (p NodeSet) XGo_text__1() (val string, err error) {
	node, err := p.XGo_first()
	if err == nil {
		for _, c := range node.Children {
			if data, ok := c.(xml.CharData); ok {
				return unsafe.String(unsafe.SliceData(data), len(data)), nil
			}
		}
		err = dql.ErrNotFound // text not found on first node
	}
	return
}

// _int retrieves the integer value from the text content of the first child
// text node. It only retrieves from the first node in the NodeSet.
func (p NodeSet) XGo_int() (int, error) {
	text, err := p.XGo_text__1()
	if err != nil {
		return 0, err
	}
	return dql.Int(text)
}

// -----------------------------------------------------------------------------
