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

package classfile

import (
	goast "go/ast"
	"strconv"
	"strings"
	"unicode"

	xast "github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/token"
)

const directivePrefix = "//xgo:class:"

const (
	directiveResource                = "resource"
	directiveResourceDiscovery       = "resource-discovery"
	directiveResourceNameDiscovery   = "resource-name-discovery"
	directiveResourceAPIScopeBinding = "resource-api-scope-binding"
)

// directives groups parsed classfile directives attached to one declaration.
type directives struct {
	resource         []directive
	discovery        []directive
	nameDiscovery    []directive
	apiScopeBindings []rawAPIScopeBinding
	invalid          []directive
}

// directive is one parsed directive with its source position and argument.
type directive struct {
	pos token.Pos
	arg string
}

// rawAPIScopeBinding is one unresolved API-position scope binding directive.
type rawAPIScopeBinding struct {
	pos    token.Pos
	target int
	source ResourceAPIScopeSource
}

// parseDirectives parses classfile directives from one comment group.
func parseDirectives(doc *xast.CommentGroup) directives {
	var ret directives
	if doc == nil {
		return ret
	}
	for _, comment := range doc.List {
		body, ok := strings.CutPrefix(comment.Text, directivePrefix)
		if !ok {
			continue
		}
		name, arg := body, ""
		if i := strings.IndexFunc(body, unicode.IsSpace); i >= 0 {
			name, arg = body[:i], strings.TrimSpace(body[i:])
		}
		pos := token.Pos(comment.Pos())
		if arg == "" {
			switch name {
			case directiveResource,
				directiveResourceDiscovery,
				directiveResourceNameDiscovery,
				directiveResourceAPIScopeBinding:
				ret.invalid = append(ret.invalid, directive{pos: pos, arg: name})
			}
			continue
		}
		switch name {
		case directiveResourceDiscovery:
			ret.discovery = append(ret.discovery, directive{
				pos: pos,
				arg: arg,
			})
		case directiveResourceNameDiscovery:
			ret.nameDiscovery = append(ret.nameDiscovery, directive{
				pos: pos,
				arg: arg,
			})
		case directiveResourceAPIScopeBinding:
			target, source, ok := parseScopeBinding(arg)
			if ok {
				ret.apiScopeBindings = append(ret.apiScopeBindings, rawAPIScopeBinding{
					pos:    pos,
					target: target,
					source: source,
				})
				continue
			}
			ret.invalid = append(ret.invalid, directive{
				pos: pos,
				arg: directiveResourceAPIScopeBinding,
			})
		case directiveResource:
			ret.resource = append(ret.resource, directive{
				pos: pos,
				arg: arg,
			})
		}
	}
	return ret
}

// validResourceKind reports whether kind is one canonical resource kind spelling.
func validResourceKind(kind string) bool {
	if kind == "" {
		return false
	}
	for seg := range strings.SplitSeq(kind, ".") {
		if seg == "" || seg[0] < 'a' || seg[0] > 'z' {
			return false
		}
		for i := 1; i < len(seg); i++ {
			ch := seg[i]
			if ch == '_' || ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'z' {
				continue
			}
			return false
		}
	}
	return true
}

// parseScopeBinding parses one target-source API-position binding directive.
func parseScopeBinding(arg string) (int, ResourceAPIScopeSource, bool) {
	const receiver = "receiver"

	fields := strings.Fields(arg)
	if len(fields) != 2 {
		return 0, ResourceAPIScopeSource{}, false
	}
	target, ok := parseParam(fields[0])
	if !ok {
		return 0, ResourceAPIScopeSource{}, false
	}
	if fields[1] == receiver {
		return target, ResourceAPIScopeSource{Receiver: true}, true
	}
	sourceParam, ok := parseParam(fields[1])
	if !ok {
		return 0, ResourceAPIScopeSource{}, false
	}
	return target, ResourceAPIScopeSource{Param: sourceParam}, true
}

// parseParam parses one param.N API-position operand.
func parseParam(v string) (int, bool) {
	const paramPrefix = "param."

	n, ok := strings.CutPrefix(v, paramPrefix)
	if !ok {
		return 0, false
	}
	if n == "" || len(n) > 1 && n[0] == '0' {
		return 0, false
	}
	ret, err := strconv.Atoi(n)
	if err != nil || ret < 0 {
		return 0, false
	}
	return ret, true
}

// xgoDeclDoc reports the effective doc group for one XGo type declaration.
func xgoDeclDoc(decl *xast.GenDecl, spec *xast.TypeSpec) *xast.CommentGroup {
	if !decl.Lparen.IsValid() && decl.Doc != nil {
		return decl.Doc
	}
	return spec.Doc
}

// goDeclDoc reports the effective doc group for one Go type declaration.
func goDeclDoc(decl *goast.GenDecl, spec *goast.TypeSpec) *xast.CommentGroup {
	if !decl.Lparen.IsValid() && decl.Doc != nil {
		return (*xast.CommentGroup)(decl.Doc)
	}
	return (*xast.CommentGroup)(spec.Doc)
}

// xgoRecvBaseName reports the receiver base type name for one XGo method.
func xgoRecvBaseName(recv *xast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	return xgoExprBaseName(recv.List[0].Type)
}

// xgoExprBaseName reports the base identifier of one XGo receiver type expression.
func xgoExprBaseName(expr xast.Expr) string {
	switch v := expr.(type) {
	case *xast.Ident:
		return v.Name
	case *xast.StarExpr:
		return xgoExprBaseName(v.X)
	case *xast.ParenExpr:
		return xgoExprBaseName(v.X)
	default:
		return ""
	}
}

// goRecvBaseName reports the receiver base type name for one Go method.
func goRecvBaseName(recv *goast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	return goExprBaseName(recv.List[0].Type)
}

// goExprBaseName reports the base identifier of one Go receiver type expression.
func goExprBaseName(expr goast.Expr) string {
	switch v := expr.(type) {
	case *goast.Ident:
		return v.Name
	case *goast.StarExpr:
		return goExprBaseName(v.X)
	case *goast.ParenExpr:
		return goExprBaseName(v.X)
	case *goast.IndexExpr:
		return goExprBaseName(v.X)
	case *goast.IndexListExpr:
		return goExprBaseName(v.X)
	default:
		return ""
	}
}
