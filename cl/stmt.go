/*
 * Copyright (c) 2021 The XGo Authors (xgo.dev). All rights reserved.
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

package cl

import (
	"fmt"
	"go/constant"
	"log"
	"path/filepath"

	goast "go/ast"
	gotoken "go/token"
	"go/types"

	"github.com/goplus/gogen"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/token"
)

func fileLineFile(relBaseDir, absFile string) string {
	if ret, err := filepath.Rel(relBaseDir, absFile); err == nil {
		return filepath.ToSlash(ret)
	}
	return absFile
}

func relFile(dir string, absFile string) string {
	if dir != "" {
		return fileLineFile(dir, absFile)
	}
	return absFile
}

func commentStmt(ctx *blockCtx, stmt ast.Stmt) {
	if ctx.fileLine {
		commentStmtEx(ctx.cb, ctx.pkgCtx, stmt)
	}
}

func commentStmtEx(cb *gogen.CodeBuilder, ctx *pkgCtx, stmt ast.Stmt) {
	start := stmt.Pos()
	if start == token.NoPos {
		cb.SetComments(nil, false)
		return
	}
	if doc := checkStmtDoc(stmt); doc != nil {
		start = doc.Pos()
	}
	pos := ctx.fset.Position(start)
	if ctx.relBaseDir != "" {
		pos.Filename = fileLineFile(ctx.relBaseDir, pos.Filename)
	}
	line := fmt.Sprintf("\n//line %s:%d:1", pos.Filename, pos.Line)
	comments := &goast.CommentGroup{
		List: []*goast.Comment{{Text: line}},
	}
	cb.SetComments(comments, false)
}

func checkStmtDoc(stmt ast.Stmt) *ast.CommentGroup {
	if decl, ok := stmt.(*ast.DeclStmt); ok {
		if d, ok := decl.Decl.(*ast.GenDecl); ok {
			return d.Doc
		}
	}
	return nil
}

func commentFunc(ctx *blockCtx, fn *gogen.Func, decl *ast.FuncDecl) {
	start := decl.Name.Pos()
	if ctx.fileLine && start != token.NoPos {
		if decl.Doc != nil {
			start = decl.Doc.Pos()
		}
		pos := ctx.fset.Position(start)
		if ctx.relBaseDir != "" {
			pos.Filename = fileLineFile(ctx.relBaseDir, pos.Filename)
		}
		var line string
		if decl.Shadow {
			line = fmt.Sprintf("//line %s:%d", pos.Filename, pos.Line)
		} else {
			line = fmt.Sprintf("//line %s:%d:1", pos.Filename, pos.Line)
		}
		doc := &goast.CommentGroup{}
		doc.List = append(doc.List, &goast.Comment{Text: line})
		if decl.Doc != nil {
			doc.List = append(doc.List, decl.Doc.List...)
		}
		fn.SetComments(ctx.pkg, doc)
	} else if decl.Doc != nil {
		fn.SetComments(ctx.pkg, decl.Doc)
	}
}

func compileStmts(ctx *blockCtx, body []ast.Stmt) {
	for _, stmt := range body {
		if v, ok := stmt.(*ast.LabeledStmt); ok {
			expr := v.Label
			ctx.cb.NewLabel(expr.Pos(), expr.Name)
		}
	}
	for _, stmt := range body {
		compileStmt(ctx, stmt)
	}
}

func compileStmt(ctx *blockCtx, stmt ast.Stmt) {
	if enableRecover {
		defer func() {
			if e := recover(); e != nil {
				ctx.handleRecover(e, stmt)
				ctx.cb.ResetStmt()
			}
		}()
	}
	commentStmt(ctx, stmt)
	switch v := stmt.(type) {
	case *ast.ExprStmt:
		x := v.X
		inFlags := checkCommandWithoutArgs(x)
		compileExpr(ctx, x, inFlags)
	case *ast.AssignStmt:
		compileAssignStmt(ctx, v)
	case *ast.ReturnStmt:
		compileReturnStmt(ctx, v)
	case *ast.IfStmt:
		compileIfStmt(ctx, v)
	case *ast.SwitchStmt:
		compileSwitchStmt(ctx, v)
	case *ast.RangeStmt:
		compileRangeStmt(ctx, v)
	case *ast.ForStmt:
		compileForStmt(ctx, v)
	case *ast.ForPhraseStmt:
		compileForPhraseStmt(ctx, v)
	case *ast.IncDecStmt:
		compileIncDecStmt(ctx, v)
	case *ast.DeferStmt:
		compileDeferStmt(ctx, v)
	case *ast.GoStmt:
		compileGoStmt(ctx, v)
	case *ast.DeclStmt:
		compileDeclStmt(ctx, v)
	case *ast.TypeSwitchStmt:
		compileTypeSwitchStmt(ctx, v)
	case *ast.SendStmt:
		compileSendStmt(ctx, v)
	case *ast.BranchStmt:
		compileBranchStmt(ctx, v)
	case *ast.LabeledStmt:
		compileLabeledStmt(ctx, v)
	case *ast.BlockStmt:
		ctx.cb.Block()
		compileStmts(ctx, v.List)
		if rec := ctx.recorder(); rec != nil {
			rec.Scope(v, ctx.cb.Scope())
		}
		ctx.cb.End()
		return
	case *ast.SelectStmt:
		compileSelectStmt(ctx, v)
	case *ast.EmptyStmt:
		// do nothing
	default:
		log.Panicf("compileStmt failed: unknown - %T\n", v)
	}
	ctx.cb.EndStmt()
}

func checkCommandWithoutArgs(x ast.Expr) int {
retry:
	if v, ok := x.(*ast.SelectorExpr); ok {
		x = v.X
		goto retry
	}
	if _, ok := x.(*ast.Ident); ok {
		return clCommandWithoutArgs
	}
	return 0
}

func compileReturnStmt(ctx *blockCtx, expr *ast.ReturnStmt) {
	var n = -1
	var results *types.Tuple
	for i, ret := range expr.Results {
		if c, ok := ret.(*ast.CompositeLit); ok && c.Type == nil {
			if n < 0 {
				results = ctx.cb.Func().Type().(*types.Signature).Results()
				n = results.Len()
			}
			var typ types.Type
			if i < n {
				typ = results.At(i).Type()
			}
			compileCompositeLit(ctx, c, typ, true)
		} else {
			inFlags := 0
			if len(expr.Results) == 1 {
				if _, ok := ret.(*ast.ComprehensionExpr); ok {
					results = ctx.cb.Func().Type().(*types.Signature).Results()
					if results.Len() == 2 {
						inFlags = clCallWithTwoValue
					}
				}
			}
			switch v := ret.(type) {
			case *ast.LambdaExpr, *ast.LambdaExpr2:
				rtyp := ctx.cb.Func().Type().(*types.Signature).Results().At(i).Type()
				sig, ok := rtyp.(*types.Signature)
				if !ok {
					panic(ctx.newCodeErrorf(
						ret.Pos(), ret.End(), "cannot use lambda expression as type %v in return statement", rtyp))
				}
				compileLambda(ctx, v, sig)
			case *ast.SliceLit:
				rtyp := ctx.cb.Func().Type().(*types.Signature).Results().At(i).Type()
				compileSliceLit(ctx, v, rtyp)
			default:
				compileExpr(ctx, ret, inFlags)
			}
		}
	}
	ctx.cb.Return(len(expr.Results), expr)
}

func compileIncDecStmt(ctx *blockCtx, expr *ast.IncDecStmt) {
	compileExprLHS(ctx, expr.X)
	ctx.cb.IncDec(gotoken.Token(expr.Tok))
}

// issue #2107:
//
//	a <- v
//	foo.a <- v
func isAppendable(x ast.Expr) bool {
	switch x := x.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		_, ok := x.X.(*ast.Ident)
		return ok
	}
	return false
}

func compileSendStmt(ctx *blockCtx, expr *ast.SendStmt) {
	cb := ctx.cb
	ch, vals := expr.Chan, expr.Values
	stk := cb.InternalStack()
	if isAppendable(ch) { // a <- v1, v2, v3 (issue #2107)
		compileExpr(ctx, ch)
		a := stk.Get(-1)
		t := a.Type.Underlying()
		if _, ok := t.(*types.Slice); ok { // a = append(a, v1, v2, v3)
			stk.Pop()
			compileExprLHS(ctx, ch)
			cb.Val(ctx.pkg.Builtin().Ref("append"))
			stk.Push(a)
			for _, v := range vals {
				compileExpr(ctx, v)
			}
			flags := gogen.InstrFlags(0)
			if expr.Ellipsis != 0 { // a = append(a, b...)
				flags |= gogen.InstrFlagEllipsis
			}
			cb.CallWith(len(vals)+1, flags, expr).AssignWith(1, 1, expr)
			return
		}
		goto normal
	}
	compileExpr(ctx, ch)

normal:
	if len(vals) != 1 || expr.Ellipsis != 0 {
		panic(ctx.newCodeError(vals[0].Pos(), vals[0].End(), "can't send multiple values to a channel"))
	}
	compileExpr(ctx, vals[0])
	ctx.cb.Send()
}

func compileAssignStmt(ctx *blockCtx, expr *ast.AssignStmt) {
	tok := expr.Tok
	inFlags := 0
	if len(expr.Lhs) == 2 && len(expr.Rhs) == 1 {
		inFlags = clCallWithTwoValue
	}
	if tok == token.DEFINE {
		stk := ctx.cb.InternalStack()
		base := stk.Len()
		names := make([]string, len(expr.Lhs))
		for i, lhs := range expr.Lhs {
			if v, ok := lhs.(*ast.Ident); ok {
				names[i] = v.Name
			} else {
				compileExprLHS(ctx, lhs) // only for typesutil.Check
				log.Panicln("TODO: non-name $v on left side of :=")
			}
		}
		if rec := ctx.recorder(); rec != nil {
			newNames := make([]*ast.Ident, 0, len(names))
			scope := ctx.cb.Scope()
			for _, lhs := range expr.Lhs {
				v := lhs.(*ast.Ident)
				if scope.Lookup(v.Name) == nil {
					newNames = append(newNames, v)
				}
			}
			defer defNames(ctx, newNames, scope)
		}
		ctx.cb.DefineVarStart(expr.Pos(), names...)
		if enableRecover {
			defer func() {
				if e := recover(); e != nil {
					ctx.cb.ResetInit()
					panic(e)
				}
			}()
		}
		for _, rhs := range expr.Rhs {
			compileExpr(ctx, rhs, inFlags)
		}
		ctx.cb.EndInit(stk.Len() - base)
		return
	}
	for _, lhs := range expr.Lhs {
		compileExprLHS(ctx, lhs)
	}
	for i, rhs := range expr.Rhs {
		switch e := unparen(rhs).(type) {
		case *ast.LambdaExpr, *ast.LambdaExpr2:
			if len(expr.Lhs) == 1 && len(expr.Rhs) == 1 {
				typ := ctx.cb.Get(-1).Type.(interface{ Elem() types.Type }).Elem()
				sig, err := checkLambdaFuncType(ctx, e, typ, clLambaAssign, expr.Lhs[0])
				if err != nil {
					panic(err)
				}
				compileLambda(ctx, e, sig)
			} else {
				panic(ctx.newCodeErrorf(e.Pos(), e.End(), "lambda unsupport multiple assignment"))
			}
		case *ast.SliceLit:
			var typ types.Type
			if len(expr.Lhs) == len(expr.Rhs) {
				typ, _ = gogen.DerefType(ctx.cb.Get(-1 - i).Type)
			}
			compileSliceLit(ctx, e, typ)
		case *ast.CompositeLit:
			var typ types.Type
			if len(expr.Lhs) == len(expr.Rhs) {
				typ, _ = gogen.DerefType(ctx.cb.Get(-1 - i).Type)
			}
			compileCompositeLit(ctx, e, typ, false)
		default:
			compileExpr(ctx, rhs, inFlags)
		}
	}
	if tok == token.ASSIGN {
		ctx.cb.AssignWith(len(expr.Lhs), len(expr.Rhs), expr)
		return
	}
	if len(expr.Lhs) != 1 || len(expr.Rhs) != 1 {
		panic("TODO: invalid syntax of assign by operator")
	}
	ctx.cb.AssignOp(gotoken.Token(tok), expr)
}

// forRange(names...) x rangeAssignThen
//
//	body
//
// end
// forRange k v x rangeAssignThen
//
//	body
//
// end
func compileRangeStmt(ctx *blockCtx, v *ast.RangeStmt) {
	if re, ok := v.X.(*ast.RangeExpr); ok {
		tok := token.DEFINE
		if v.Tok == token.ASSIGN {
			tok = v.Tok
		}
		compileForStmt(ctx, toForStmt(v.For, v.Key, v.Body, re, tok, nil))
		return
	}
	cb := ctx.cb
	defer cb.End(v)
	defer func() {
		r := recover()
		if r != nil {
			ctx.handleRecover(r, v)
			cb.ResetStmt()
		}
	}()
	comments, once := cb.BackupComments()
	defineNames := make([]*ast.Ident, 0, 2)
	if v.Tok == token.DEFINE {
		names := make([]string, 1, 2)
		if v.Key == nil {
			names[0] = "_"
		} else {
			key := v.Key.(*ast.Ident)
			names[0] = key.Name
			defineNames = append(defineNames, key)
		}
		if v.Value != nil {
			value := v.Value.(*ast.Ident)
			names = append(names, value.Name)
			defineNames = append(defineNames, value)
		}
		cb.ForRangeEx(names, v)
		compileExpr(ctx, v.X)
	} else {
		cb.ForRangeEx(nil, v)
		n := 0
		if v.Key == nil {
			if v.Value != nil {
				ctx.cb.VarRef(nil) // underscore
				n++
			}
		} else {
			compileExprLHS(ctx, v.Key)
			n++
		}
		if v.Value != nil {
			compileExprLHS(ctx, v.Value)
			n++
		}
		compileExpr(ctx, v.X)
	}
	pos := v.TokPos
	if pos == 0 {
		pos = v.For
	}
	cb.RangeAssignThen(pos)
	if len(defineNames) > 0 {
		defNames(ctx, defineNames, cb.Scope())
	}
	if rec := ctx.recorder(); rec != nil {
		rec.Scope(v, cb.Scope())
	}
	cb.VBlock()
	compileStmts(ctx, v.Body.List)
	if rec := ctx.recorder(); rec != nil {
		rec.Scope(v.Body, cb.Scope())
	}
	cb.End(v.Body)
	cb.SetComments(comments, once)
	setBodyHandler(ctx)
}

func compileForPhraseStmt(ctx *blockCtx, v *ast.ForPhraseStmt) {
	if re, ok := v.X.(*ast.RangeExpr); ok {
		compileForStmt(ctx, toForStmt(v.For, v.Value, v.Body, re, token.DEFINE, v.ForPhrase))
		return
	}
	cb := ctx.cb
	defer cb.End(v)
	defer func() {
		r := recover()
		if r != nil {
			ctx.handleRecover(r, v)
			cb.ResetStmt()
		}
	}()
	comments, once := cb.BackupComments()
	names := make([]string, 1, 2)
	defineNames := make([]*ast.Ident, 0, 2)
	if v.Key == nil {
		names[0] = "_"
	} else {
		names[0] = v.Key.Name
		defineNames = append(defineNames, v.Key)
	}
	if v.Value != nil {
		names = append(names, v.Value.Name)
		defineNames = append(defineNames, v.Value)
	}
	cb.ForRange(names...)
	compileExpr(ctx, v.X)
	cb.RangeAssignThen(v.TokPos)
	if len(defineNames) > 0 {
		defNames(ctx, defineNames, cb.Scope())
	}
	if rec := ctx.recorder(); rec != nil {
		rec.Scope(v, cb.Scope())
	}
	if v.Cond != nil {
		cb.If()
		compileExpr(ctx, v.Cond)
		cb.Then()
		compileStmts(ctx, v.Body.List)
		cb.SetComments(comments, once)
		if rec := ctx.recorder(); rec != nil {
			rec.Scope(v.Body, cb.Scope())
		}
		cb.End()
	} else {
		cb.VBlock()
		compileStmts(ctx, v.Body.List)
		if rec := ctx.recorder(); rec != nil {
			rec.Scope(v.Body, cb.Scope())
		}
		cb.End(v.Body)
		cb.SetComments(comments, once)
	}
	setBodyHandler(ctx)
}

func toForStmt(forPos token.Pos, value ast.Expr, body *ast.BlockStmt, re *ast.RangeExpr, tok token.Token, fp *ast.ForPhrase) *ast.ForStmt {
	const (
		nameK    = "_xgo_k"
		nameStep = "_xgo_step"
		nameEnd  = "_xgo_end"
	)
	nilIdent := value == nil
	if !nilIdent {
		if v, ok := value.(*ast.Ident); ok && v.Name == "_" {
			nilIdent = true
		}
	}
	if nilIdent {
		value = &ast.Ident{NamePos: forPos, Name: nameK}
	}
	first := re.First
	if first == nil {
		first = &ast.BasicLit{ValuePos: forPos, Kind: token.INT, Value: "0"}
	}
	initLhs := []ast.Expr{value}
	initRhs := []ast.Expr{first}
	replaceValue := false
	var cond ast.Expr
	var post ast.Expr
	switch re.Last.(type) {
	case *ast.Ident, *ast.BasicLit:
		cond = re.Last
	default:
		replaceValue = true
		cond = &ast.Ident{NamePos: forPos, Name: nameEnd}
		initLhs = append(initLhs, cond)
		initRhs = append(initRhs, re.Last)
	}
	if re.Expr3 == nil {
		post = &ast.BasicLit{ValuePos: forPos, Kind: token.INT, Value: "1"}
	} else {
		switch re.Expr3.(type) {
		case *ast.Ident, *ast.BasicLit:
			post = re.Expr3
		default:
			replaceValue = true
			post = &ast.Ident{NamePos: forPos, Name: nameStep}
			initLhs = append(initLhs, post)
			initRhs = append(initRhs, re.Expr3)
		}
	}
	if tok == token.ASSIGN && replaceValue {
		oldValue := value
		value = &ast.Ident{NamePos: forPos, Name: nameK}
		initLhs[0] = value
		body.List = append([]ast.Stmt{&ast.AssignStmt{
			Lhs:    []ast.Expr{oldValue},
			TokPos: forPos,
			Tok:    token.ASSIGN,
			Rhs:    []ast.Expr{value},
		}}, body.List...)
		tok = token.DEFINE
	}
	if fp != nil && fp.Cond != nil {
		condStmt := &ast.IfStmt{
			If:   fp.IfPos,
			Init: fp.Init,
			Cond: fp.Cond,
			Body: body,
		}
		body = &ast.BlockStmt{
			List: []ast.Stmt{condStmt},
		}
	}
	return &ast.ForStmt{
		For: forPos,
		Init: &ast.AssignStmt{
			Lhs:    initLhs,
			TokPos: re.To,
			Tok:    tok,
			Rhs:    initRhs,
		},
		Cond: &ast.BinaryExpr{
			X:     value,
			OpPos: re.To,
			Op:    token.LSS,
			Y:     cond,
		},
		Post: &ast.AssignStmt{
			Lhs:    []ast.Expr{value},
			TokPos: re.Colon2,
			Tok:    token.ADD_ASSIGN,
			Rhs:    []ast.Expr{post},
		},
		Body: body,
	}
}

// for init; cond then
//
//	body
//	post
//
// end
func compileForStmt(ctx *blockCtx, v *ast.ForStmt) {
	cb := ctx.cb
	defer cb.End(v)
	defer func() {
		r := recover()
		if r != nil {
			ctx.handleRecover(r, v)
			cb.ResetStmt()
		}
	}()
	comments, once := cb.BackupComments()
	cb.For(v)
	if rec := ctx.recorder(); rec != nil {
		rec.Scope(v, cb.Scope())
	}
	if v.Init != nil {
		compileStmt(ctx, v.Init)
	}
	if v.Cond != nil {
		compileExpr(ctx, v.Cond)
	} else {
		cb.None()
	}
	cb.Then(v.Body)
	if rec := ctx.recorder(); rec != nil {
		rec.Scope(v.Body, cb.Scope())
	}
	compileStmts(ctx, v.Body.List)
	if v.Post != nil {
		cb.Post()
		compileStmt(ctx, v.Post)
	}
	cb.SetComments(comments, once)
	setBodyHandler(ctx)
}

// if init; cond then
//
//	body
//
// end
func compileIfStmt(ctx *blockCtx, v *ast.IfStmt) {
	cb := ctx.cb
	defer cb.End(v)
	defer func() {
		r := recover()
		if r != nil {
			ctx.handleRecover(r, v)
			cb.ResetStmt()
		}
	}()
	comments, once := cb.BackupComments()
	cb.If(v)
	if v.Init != nil {
		compileStmt(ctx, v.Init)
	}
	compileExpr(ctx, v.Cond)
	if rec := ctx.recorder(); rec != nil {
		rec.Scope(v, cb.Scope())
	}
	cb.Then(v.Body)
	compileStmts(ctx, v.Body.List)
	if e := v.Else; e != nil {
		cb.Else(e)
		if stmts, ok := e.(*ast.BlockStmt); ok {
			compileStmts(ctx, stmts.List)
		} else {
			compileStmt(ctx, e)
		}
	}
	cb.SetComments(comments, once)
	if rec := ctx.recorder(); rec != nil {
		rec.Scope(v.Body, cb.Scope())
	}
}

// typeSwitch(name) init; expr typeAssertThen()
// type1 type2 ... typeN typeCase(N)
//
//	...
//	end
//
// type1 type2 ... typeM typeCase(M)
//
//	...
//	end
//
// end
func compileTypeSwitchStmt(ctx *blockCtx, v *ast.TypeSwitchStmt) {
	var cb = ctx.cb
	defer cb.End(v)
	defer func() {
		r := recover()
		if r != nil {
			ctx.handleRecover(r, v)
			cb.ResetStmt()
		}
	}()
	comments, once := cb.BackupComments()
	var name string
	var ta *ast.TypeAssertExpr
	switch stmt := v.Assign.(type) {
	case *ast.AssignStmt:
		if stmt.Tok != token.DEFINE || len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
			panic("TODO: type switch syntax error")
		}
		name = stmt.Lhs[0].(*ast.Ident).Name
		ta = stmt.Rhs[0].(*ast.TypeAssertExpr)
	case *ast.ExprStmt:
		ta = stmt.X.(*ast.TypeAssertExpr)
	}
	if ta.Type != nil {
		panic("TODO: type switch syntax error, please use x.(type)")
	}
	cb.TypeSwitch(name, v)
	if rec := ctx.recorder(); rec != nil {
		rec.Scope(v, cb.Scope())
	}
	if v.Init != nil {
		compileStmt(ctx, v.Init)
	}
	compileExpr(ctx, ta.X)
	cb.TypeAssertThen()
	seen := make(map[types.Type]ast.Expr)
	var firstDefault ast.Stmt
	for _, stmt := range v.Body.List {
		c, ok := stmt.(*ast.CaseClause)
		if !ok {
			log.Panicln("TODO: compile TypeSwitchStmt failed - case clause expected.")
		}
		cb.TypeCase(c)
		for _, citem := range c.List {
			compileExpr(ctx, citem)
			T := cb.Get(-1).Type
			if tt, ok := T.(*gogen.TypeType); ok {
				T = tt.Type()
			}
			var haserr bool
			for t, other := range seen {
				if T == nil && t == nil || T != nil && t != nil && types.Identical(T, t) {
					haserr = true
					pos := citem.Pos()
					end := citem.End()
					if T == types.Typ[types.UntypedNil] {
						ctx.handleErrorf(
							pos, end, "multiple nil cases in type switch (first at %v)",
							ctx.Position(other.Pos()))
					} else {
						ctx.handleErrorf(
							pos, end, "duplicate case %s in type switch\n\tprevious case at %v",
							T, ctx.Position(other.Pos()))
					}
				}
			}
			if !haserr {
				seen[T] = citem
			}
		}
		if c.List == nil {
			if firstDefault != nil {
				ctx.handleErrorf(
					c.Pos(), c.End(), "multiple defaults in type switch (first at %v)", ctx.Position(firstDefault.Pos()))
			} else {
				firstDefault = c
			}
		}
		cb.Then()
		compileStmts(ctx, c.Body)
		commentStmt(ctx, stmt)
		if rec := ctx.recorder(); rec != nil {
			rec.Scope(c, cb.Scope())
		}
		cb.End(c)
	}
	cb.SetComments(comments, once)
}

// switch init; tag then
// expr1 expr2 ... exprN case(N)
//
//	...
//	end
//
// expr1 expr2 ... exprM case(M)
//
//	...
//	end
//
// end
func compileSwitchStmt(ctx *blockCtx, v *ast.SwitchStmt) {
	cb := ctx.cb
	defer cb.End(v)
	defer func() {
		r := recover()
		if r != nil {
			ctx.handleRecover(r, v)
			cb.ResetStmt()
		}
	}()
	comments, once := cb.BackupComments()
	cb.Switch(v)
	if v.Init != nil {
		compileStmt(ctx, v.Init)
	}
	if v.Tag != nil { // switch tag {....}
		compileExpr(ctx, v.Tag)
	} else {
		cb.None() // switch {...}
	}
	if rec := ctx.recorder(); rec != nil {
		rec.Scope(v, cb.Scope())
	}
	cb.Then(v.Body)
	seen := make(valueMap)
	var firstDefault ast.Stmt
	for _, stmt := range v.Body.List {
		c, ok := stmt.(*ast.CaseClause)
		if !ok {
			log.Panicln("TODO: compile SwitchStmt failed - case clause expected.")
		}
		cb.Case(c)
		for _, citem := range c.List {
			compileExpr(ctx, citem)
			v := cb.Get(-1)
			if val := goVal(v.CVal); val != nil {
				// look for duplicate types for a given value
				// (quadratic algorithm, but these lists tend to be very short)
				typ := types.Default(v.Type)
				var haserr bool
				for _, vt := range seen[val] {
					if types.Identical(typ, vt.typ) {
						haserr = true
						src := ctx.LoadExpr(v.Src)
						if lit, ok := v.Src.(*ast.BasicLit); ok {
							ctx.handleErrorf(lit.Pos(), lit.End(), "duplicate case %s in switch\n\tprevious case at %v",
								src, ctx.Position(vt.pos))
						} else {
							ctx.handleErrorf(v.Src.Pos(), v.Src.End(), "duplicate case %s (value %#v) in switch\n\tprevious case at %v",
								src, val, ctx.Position(vt.pos))
						}
					}
				}
				if !haserr {
					seen[val] = append(seen[val], valueType{v.Src.Pos(), typ})
				}
			}
		}
		if c.List == nil {
			if firstDefault != nil {
				ctx.handleErrorf(c.Pos(), c.End(), "multiple defaults in switch (first at %v)", ctx.Position(firstDefault.Pos()))
			} else {
				firstDefault = c
			}
		}
		cb.Then()
		body, has := hasFallthrough(c.Body)
		compileStmts(ctx, body)
		if has {
			cb.Fallthrough()
		}
		commentStmt(ctx, stmt)
		if rec := ctx.recorder(); rec != nil {
			rec.Scope(c, cb.Scope())
		}
		cb.End(c)
	}
	cb.SetComments(comments, once)
}

func hasFallthrough(body []ast.Stmt) ([]ast.Stmt, bool) {
	if n := len(body); n > 0 {
		if bs, ok := body[n-1].(*ast.BranchStmt); ok && bs.Tok == token.FALLTHROUGH {
			return body[:n-1], true
		}
	}
	return body, false
}

// select
// stmt1 commCase(1)
//
//	...
//	end
//
// stmt2 commCase(1)
//
//	...
//	end
//
// ...
// commCase(0)
//
//	...
//	end
//
// end
func compileSelectStmt(ctx *blockCtx, v *ast.SelectStmt) {
	cb := ctx.cb
	defer cb.End(v)
	defer func() {
		r := recover()
		if r != nil {
			ctx.handleRecover(r, v)
			cb.ResetStmt()
		}
	}()
	comments, once := cb.BackupComments()
	cb.Select(v)
	for _, stmt := range v.Body.List {
		c, ok := stmt.(*ast.CommClause)
		if !ok {
			log.Panicln("TODO: compile SelectStmt failed - comm clause expected.")
		}
		cb.CommCase(c)
		if c.Comm != nil {
			compileStmt(ctx, c.Comm)
		}
		cb.Then()
		compileStmts(ctx, c.Body)
		commentStmt(ctx, stmt)
		if rec := ctx.recorder(); rec != nil {
			rec.Scope(c, cb.Scope())
		}
		cb.End(c)
	}
	cb.SetComments(comments, once)
}

func compileBranchStmt(ctx *blockCtx, v *ast.BranchStmt) {
	label := v.Label
	switch v.Tok {
	case token.GOTO:
		cb := ctx.cb
		if l, ok := cb.LookupLabel(label.Name); ok {
			cb.Goto(l)
			return
		}
		compileCallExpr(ctx, &ast.CallExpr{
			Fun:        &ast.Ident{NamePos: v.TokPos, Name: "goto", Obj: &ast.Object{Data: label}},
			Args:       []ast.Expr{label},
			NoParenEnd: label.End(),
		}, clIdentGoto)
	case token.BREAK:
		ctx.cb.Break(getLabel(ctx, label))
	case token.CONTINUE:
		ctx.cb.Continue(getLabel(ctx, label))
	case token.FALLTHROUGH:
		ctx.handleErrorf(v.Pos(), v.End(), "fallthrough statement out of place")
	default:
		panic("unknown branch statement")
	}
}

func getLabel(ctx *blockCtx, label *ast.Ident) *gogen.Label {
	if label != nil {
		if l, ok := ctx.cb.LookupLabel(label.Name); ok {
			return l
		}
		ctx.handleErrorf(label.Pos(), label.End(), "label %v is not defined", label.Name)
	}
	return nil
}

func compileLabeledStmt(ctx *blockCtx, v *ast.LabeledStmt) {
	l, _ := ctx.cb.LookupLabel(v.Label.Name)
	ctx.cb.Label(l)
	compileStmt(ctx, v.Stmt)
}

func compileGoStmt(ctx *blockCtx, v *ast.GoStmt) {
	compileCallExpr(ctx, v.Call, 0)
	ctx.cb.Go()
}

func compileDeferStmt(ctx *blockCtx, v *ast.DeferStmt) {
	compileCallExpr(ctx, v.Call, 0)
	ctx.cb.Defer()
}

func compileDeclStmt(ctx *blockCtx, expr *ast.DeclStmt) {
	switch d := expr.Decl.(type) {
	case *ast.GenDecl:
		switch d.Tok {
		case token.TYPE:
			for _, spec := range d.Specs {
				compileType(ctx, spec.(*ast.TypeSpec))
			}
		case token.CONST:
			cdecl := ctx.pkg.NewConstDefs(ctx.cb.Scope())
			loadConstSpecs(ctx, cdecl, d.Specs)
		case token.VAR:
			for _, spec := range d.Specs {
				v := spec.(*ast.ValueSpec)
				loadVars(ctx, v, d.Doc, false)
			}
		default:
			log.Panicln("TODO: compileDeclStmt - unknown")
		}
	}
}

func compileType(ctx *blockCtx, t *ast.TypeSpec) {
	name := t.Name.Name
	if name == "_" {
		return
	}
	if t.Assign != token.NoPos { // alias type
		ctx.cb.AliasType(name, toType(ctx, t.Type))
	} else {
		ctx.cb.NewType(name).InitType(ctx.pkg, toType(ctx, t.Type))
	}
}

type (
	valueMap  map[any][]valueType // underlying Go value -> valueType
	valueType struct {
		pos token.Pos
		typ types.Type
	}
)

// goVal returns the Go value for val, or nil.
func goVal(val constant.Value) any {
	// val should exist, but be conservative and check
	if val == nil {
		return nil
	}
	// Match implementation restriction of other compilers.
	// gc only checks duplicates for integer, floating-point
	// and string values, so only create Go values for these
	// types.
	switch val.Kind() {
	case constant.Int:
		if x, ok := constant.Int64Val(val); ok {
			return x
		}
		if x, ok := constant.Uint64Val(val); ok {
			return x
		}
	case constant.Float:
		if x, ok := constant.Float64Val(val); ok {
			return x
		}
	case constant.String:
		return constant.StringVal(val)
	}
	return nil
}

// -----------------------------------------------------------------------------
