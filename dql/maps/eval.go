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

package maps

import (
	"fmt"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/token"
)

type evalContext struct {
	root NodeSet
	self *Node
}

type compiledQuery struct {
	expr ast.Expr
	err  error
}

var evalCache sync.Map

// Eval evaluates query as a DQL query fragment relative to the implicit root
// created from src.
func Eval(query string, src any) (NodeSet, error) {
	expr, err := compileQuery(query)
	if err != nil {
		return NodeSet{}, err
	}
	root, err := sourceForEval(src)
	if err != nil {
		return NodeSet{}, err
	}
	return evalNodeSetExpr(expr, evalContext{root: root})
}

func compileQuery(query string) (ast.Expr, error) {
	if cached, ok := evalCache.Load(query); ok {
		compiled := cached.(compiledQuery)
		return compiled.expr, compiled.err
	}
	expr, err := parser.ParseExpr(query)
	compiled := compiledQuery{expr: expr, err: err}
	actual, loaded := evalCache.LoadOrStore(query, compiled)
	if loaded {
		compiled = actual.(compiledQuery)
	}
	return compiled.expr, compiled.err
}

func sourceForEval(src any) (ret NodeSet, err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("dql/maps.Eval: %v", v)
		}
	}()
	ret = Source(src)
	return ret, nil
}

func evalNodeSetExpr(expr ast.Expr, ctx evalContext) (NodeSet, error) {
	switch x := expr.(type) {
	case *ast.ParenExpr:
		return evalNodeSetExpr(x.X, ctx)
	case *ast.Ident:
		if x.Name == "self" {
			if ctx.self == nil {
				return NodeSet{}, fmt.Errorf("dql/maps.Eval: self is unavailable here")
			}
			return Root(*ctx.self), nil
		}
		return ctx.root.XGo_Elem(unquoteName(x.Name)), nil
	case *ast.BasicLit:
		if x.Kind != token.STRING {
			return NodeSet{}, fmt.Errorf("dql/maps.Eval: unsupported literal %q in node-set query", x.Value)
		}
		name, err := strconv.Unquote(x.Value)
		if err != nil {
			return NodeSet{}, err
		}
		return ctx.root.XGo_Elem(name), nil
	case *ast.SelectorExpr:
		base, err := evalNodeSetExpr(x.X, ctx)
		if err != nil {
			return NodeSet{}, err
		}
		switch x.Sel.Name {
		case "*":
			return base.XGo_Child(), nil
		case "all", "_all":
			return base.XGo_all(), nil
		case "one", "_one":
			return base.XGo_one(), nil
		case "single", "_single":
			return base.XGo_single(), nil
		case "ok", "_ok", "name", "_name", "value", "_value":
			return NodeSet{}, fmt.Errorf("dql/maps.Eval: selector %q yields a scalar value", x.Sel.Name)
		default:
			if strings.HasPrefix(x.Sel.Name, "$") {
				return NodeSet{}, fmt.Errorf("dql/maps.Eval: selector %q yields a scalar value", x.Sel.Name)
			}
			return base.XGo_Elem(unquoteName(x.Sel.Name)), nil
		}
	case *ast.AnySelectorExpr:
		base, err := evalNodeSetExpr(x.X, ctx)
		if err != nil {
			return NodeSet{}, err
		}
		name := unquoteName(x.Sel.Name)
		if name == "*" {
			name = ""
		}
		return base.XGo_Any(name), nil
	case *ast.CondExpr:
		base, err := evalNodeSetExpr(x.X, ctx)
		if err != nil {
			return NodeSet{}, err
		}
		if ident, ok := x.Cond.(*ast.Ident); ok {
			return base.XGo_Select(unquoteName(ident.Name)), nil
		}
		return filterNodeSet(base, x.Cond)
	case *ast.IndexExpr:
		base, err := evalNodeSetExpr(x.X, ctx)
		if err != nil {
			return NodeSet{}, err
		}
		index, err := evalIndexExpr(x.Index)
		if err != nil {
			return NodeSet{}, err
		}
		return base.XGo_Index(index), nil
	default:
		return NodeSet{}, fmt.Errorf("dql/maps.Eval: unsupported node-set expression %T", expr)
	}
}

func filterNodeSet(base NodeSet, cond ast.Expr) (NodeSet, error) {
	if base.Err != nil {
		return base, nil
	}
	nodes := make([]Node, 0, 8)
	var filterErr error
	base.Data(func(node Node) bool {
		ok, err := evalBoolExpr(cond, evalContext{root: Root(node), self: &node})
		if err != nil {
			filterErr = err
			return false
		}
		if ok {
			nodes = append(nodes, node)
		}
		return true
	})
	if filterErr != nil {
		return NodeSet{}, filterErr
	}
	return Nodes(nodes...), nil
}

func evalIndexExpr(expr ast.Expr) (int, error) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return 0, fmt.Errorf("dql/maps.Eval: unsupported index expression %T", expr)
	}
	return strconv.Atoi(lit.Value)
}

func evalBoolExpr(expr ast.Expr, ctx evalContext) (bool, error) {
	switch x := expr.(type) {
	case *ast.ParenExpr:
		return evalBoolExpr(x.X, ctx)
	case *ast.UnaryExpr:
		if x.Op != token.NOT {
			return false, fmt.Errorf("dql/maps.Eval: unsupported unary operator %s", x.Op)
		}
		ret, err := evalBoolExpr(x.X, ctx)
		if err != nil {
			return false, err
		}
		return !ret, nil
	case *ast.BinaryExpr:
		switch x.Op {
		case token.LAND:
			left, err := evalBoolExpr(x.X, ctx)
			if err != nil || !left {
				return left, err
			}
			right, err := evalBoolExpr(x.Y, ctx)
			return left && right, err
		case token.LOR:
			left, err := evalBoolExpr(x.X, ctx)
			if err != nil || left {
				return left, err
			}
			right, err := evalBoolExpr(x.Y, ctx)
			return right, err
		default:
			left, err := evalValueExpr(x.X, ctx)
			if err != nil {
				return false, err
			}
			right, err := evalValueExpr(x.Y, ctx)
			if err != nil {
				return false, err
			}
			return compareValues(x.Op, left, right)
		}
	default:
		value, err := evalValueExpr(expr, ctx)
		if err != nil {
			return false, err
		}
		ret, ok := value.(bool)
		if !ok {
			return false, fmt.Errorf("dql/maps.Eval: expression %T is not boolean", expr)
		}
		return ret, nil
	}
}

func evalValueExpr(expr ast.Expr, ctx evalContext) (any, error) {
	switch x := expr.(type) {
	case *ast.ParenExpr:
		return evalValueExpr(x.X, ctx)
	case *ast.BasicLit:
		return basicLiteralValue(x)
	case *ast.EnvExpr:
		if ctx.self == nil {
			return nil, fmt.Errorf("dql/maps.Eval: %s is unavailable here", "$"+x.Name.Name)
		}
		return ctx.self.XGo_Attr__0(unquoteName(x.Name.Name)), nil
	case *ast.Ident:
		switch x.Name {
		case "true":
			return true, nil
		case "false":
			return false, nil
		case "nil":
			return nil, nil
		default:
			return nil, fmt.Errorf("dql/maps.Eval: unsupported value identifier %q", x.Name)
		}
	case *ast.SelectorExpr:
		base, err := evalNodeSetExpr(x.X, ctx)
		if err != nil {
			return nil, err
		}
		switch x.Sel.Name {
		case "name", "_name":
			return base.XGo_name__0(), nil
		case "value", "_value":
			return base.XGo_value__0(), nil
		case "ok", "_ok":
			return base.XGo_ok(), nil
		default:
			if strings.HasPrefix(x.Sel.Name, "$") {
				return base.XGo_Attr__0(unquoteName(strings.TrimPrefix(x.Sel.Name, "$"))), nil
			}
			return nil, fmt.Errorf("dql/maps.Eval: unsupported value selector %q", x.Sel.Name)
		}
	case *ast.CallExpr:
		return evalCallExpr(x, ctx)
	case *ast.UnaryExpr, *ast.BinaryExpr:
		return evalBoolExpr(expr, ctx)
	default:
		return nil, fmt.Errorf("dql/maps.Eval: unsupported value expression %T", expr)
	}
}

func evalCallExpr(call *ast.CallExpr, ctx evalContext) (any, error) {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		switch fun.Name {
		case "match":
			if len(call.Args) != 2 {
				return nil, fmt.Errorf("dql/maps.Eval: match expects 2 arguments")
			}
			pattern, ok, err := evalStringExpr(call.Args[0], ctx)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("dql/maps.Eval: match pattern must be a string")
			}
			value, ok, err := evalStringExpr(call.Args[1], ctx)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("dql/maps.Eval: match value must be a string")
			}
			return path.Match(pattern, value)
		default:
			return nil, fmt.Errorf("dql/maps.Eval: unsupported call %q", fun.Name)
		}
	case *ast.SelectorExpr:
		receiver, ok, err := evalStringExpr(fun.X, ctx)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("dql/maps.Eval: selector call receiver must be a string")
		}
		if len(call.Args) != 1 {
			return nil, fmt.Errorf("dql/maps.Eval: selector call %q expects 1 argument", fun.Sel.Name)
		}
		arg, ok, err := evalStringExpr(call.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("dql/maps.Eval: selector call argument for %q must be a string", fun.Sel.Name)
		}
		switch fun.Sel.Name {
		case "contains":
			return strings.Contains(receiver, arg), nil
		case "hasPrefix":
			return strings.HasPrefix(receiver, arg), nil
		case "hasSuffix":
			return strings.HasSuffix(receiver, arg), nil
		default:
			return nil, fmt.Errorf("dql/maps.Eval: unsupported selector call %q", fun.Sel.Name)
		}
	default:
		return nil, fmt.Errorf("dql/maps.Eval: unsupported call expression %T", call.Fun)
	}
}

func evalStringExpr(expr ast.Expr, ctx evalContext) (string, bool, error) {
	value, err := evalValueExpr(expr, ctx)
	if err != nil {
		return "", false, err
	}
	value = zeroLike(value, "")
	text, ok := value.(string)
	return text, ok, nil
}

func basicLiteralValue(lit *ast.BasicLit) (any, error) {
	switch lit.Kind {
	case token.STRING:
		return strconv.Unquote(lit.Value)
	case token.INT:
		return strconv.Atoi(lit.Value)
	case token.FLOAT:
		return strconv.ParseFloat(lit.Value, 64)
	default:
		return nil, fmt.Errorf("dql/maps.Eval: unsupported literal kind %s", lit.Kind)
	}
}

func compareValues(op token.Token, left, right any) (bool, error) {
	left = zeroLike(left, right)
	right = zeroLike(right, left)
	if leftNumber, ok := asFloat(left); ok {
		rightNumber, ok := asFloat(right)
		if !ok {
			return false, fmt.Errorf("dql/maps.Eval: cannot compare number with %T", right)
		}
		switch op {
		case token.EQL:
			return leftNumber == rightNumber, nil
		case token.NEQ:
			return leftNumber != rightNumber, nil
		case token.GTR:
			return leftNumber > rightNumber, nil
		case token.GEQ:
			return leftNumber >= rightNumber, nil
		case token.LSS:
			return leftNumber < rightNumber, nil
		case token.LEQ:
			return leftNumber <= rightNumber, nil
		default:
			return false, fmt.Errorf("dql/maps.Eval: unsupported comparison operator %s", op)
		}
	}
	switch left := left.(type) {
	case string:
		right, ok := right.(string)
		if !ok {
			return false, fmt.Errorf("dql/maps.Eval: cannot compare string with %T", right)
		}
		switch op {
		case token.EQL:
			return left == right, nil
		case token.NEQ:
			return left != right, nil
		case token.GTR:
			return left > right, nil
		case token.GEQ:
			return left >= right, nil
		case token.LSS:
			return left < right, nil
		case token.LEQ:
			return left <= right, nil
		default:
			return false, fmt.Errorf("dql/maps.Eval: unsupported comparison operator %s", op)
		}
	case bool:
		right, ok := right.(bool)
		if !ok {
			return false, fmt.Errorf("dql/maps.Eval: cannot compare bool with %T", right)
		}
		switch op {
		case token.EQL:
			return left == right, nil
		case token.NEQ:
			return left != right, nil
		default:
			return false, fmt.Errorf("dql/maps.Eval: unsupported comparison operator %s for bool", op)
		}
	case nil:
		switch op {
		case token.EQL:
			return right == nil, nil
		case token.NEQ:
			return right != nil, nil
		default:
			return false, fmt.Errorf("dql/maps.Eval: unsupported comparison operator %s for nil", op)
		}
	default:
		switch op {
		case token.EQL:
			return reflect.DeepEqual(left, right), nil
		case token.NEQ:
			return !reflect.DeepEqual(left, right), nil
		default:
			return false, fmt.Errorf("dql/maps.Eval: unsupported comparison between %T and %T", left, right)
		}
	}
}

func asFloat(v any) (float64, bool) {
	switch v := v.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func zeroLike(v, other any) any {
	if v != nil {
		return v
	}
	switch other.(type) {
	case string:
		return ""
	case bool:
		return false
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return float64(0)
	default:
		return nil
	}
}

func unquoteName(name string) string {
	if unquoted, err := strconv.Unquote(name); err == nil {
		return unquoted
	}
	return name
}
