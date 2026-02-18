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

	"github.com/goplus/xgo/dql"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// -----------------------------------------------------------------------------

const (
	spaces = " \t\r\n"
)

// textOf returns text data of all node's children.
func textOf(node *html.Node) string {
	var p textPrinter
	p.printNode(node)
	return string(p.data)
}

type textPrinter struct {
	data         []byte
	notLineStart bool
	hasSpace     bool
}

func (p *textPrinter) printText(v string, hasRightSpace bool) {
	if v == "" {
		return
	}
	if p.notLineStart && p.hasSpace {
		p.data = append(p.data, ' ')
	} else {
		p.notLineStart = true
	}
	p.data = append(p.data, v...)
	p.hasSpace = hasRightSpace
}

func (p *textPrinter) printNode(node *html.Node) {
	if node == nil {
		return
	}
	if node.Type == html.TextNode {
		p.printText(textTrimRight(textTrimLeft(node.Data, &p.hasSpace)))
		return
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		p.printNode(child)
	}
	switch node.DataAtom {
	case atom.P, atom.Div, atom.Br, atom.H1, atom.H2, atom.H3, atom.H4,
		atom.H5, atom.H6, atom.Li, atom.Blockquote, atom.Pre:
		p.data = append(p.data, '\n')
		p.notLineStart = false
	}
}

func textTrimLeft(v string, hasSpace *bool) string {
	ret := strings.TrimLeft(v, spaces)
	if len(v) != len(ret) {
		*hasSpace = true
	}
	return ret
}

func textTrimRight(v string) (string, bool) {
	ret := strings.TrimRight(v, spaces)
	return ret, len(v) != len(ret)
}

// -----------------------------------------------------------------------------

// Text retrieves the text content of the NodeSet. It only retrieves from the
// first node in the NodeSet. It ignores any error and returns an empty string
// if there is an error.
func (p NodeSet) Text__0() string {
	val, _ := p.Text__1()
	return val
}

// Text retrieves the text content of the NodeSet. It only retrieves from the
// first node in the NodeSet.
func (p NodeSet) Text__1() (val string, err error) {
	node, err := p.First()
	if err == nil {
		val = textOf(node)
	}
	return
}

// Int retrieves the integer value from the text content of the first node in
// the NodeSet.
func (p NodeSet) Int() (int, error) {
	text, err := p.Text__1()
	if err != nil {
		return 0, err
	}
	return dql.Int(text)
}

// -----------------------------------------------------------------------------
