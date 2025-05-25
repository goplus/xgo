/*
 * Copyright (c) 2022 The XGo Authors (xgo.dev). All rights reserved.
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

package togo

import (
	"bytes"
	"go/format"
	"go/token"
	"testing"

	gopast "github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/parser"
)

func testAST(t *testing.T, from, to string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "foo.go", from, 0)
	if err != nil {
		t.Fatal("parser.ParseFile:", err)
	}
	gopf := ASTFile(f, 0)
	var b bytes.Buffer
	err = format.Node(&b, fset, gopf)
	if err != nil {
		t.Fatal("format.Node:", err)
	}
	result := b.String()
	if to != result {
		t.Fatalf("\nResult:\n%s\nExpected:\n%s\n", result, to)
	}
}

func test(t *testing.T, src string) {
	testAST(t, src, src)
}

func testPanic(t *testing.T, panicMsg string, doPanic func()) {
	t.Run(panicMsg, func(t *testing.T) {
		defer func() {
			if e := recover(); e == nil {
				t.Fatal("testPanic: no error?")
			} else if msg := e.(string); msg != panicMsg {
				t.Fatalf("\nResult:\n%s\nExpected Panic:\n%s\n", msg, panicMsg)
			}
		}()
		doPanic()
	})
}

func TestErrASTFile(t *testing.T) {
	testPanic(t, "ASTFile: doesn't support keeping func body now\n", func() {
		ASTFile(nil, KeepFuncBody)
	})
}

func TestErrDecl(t *testing.T) {
	testPanic(t, "goDecl: unknown decl - <nil>\n", func() {
		goDecl(nil)
	})
	testPanic(t, "goGenDecl: unknown spec - ILLEGAL\n", func() {
		goGenDecl(&gopast.GenDecl{
			Specs: []gopast.Spec{nil},
		})
	})
}

func TestErrExpr(t *testing.T) {
	testPanic(t, "goExpr: unknown expr - *ast.BadExpr\n", func() {
		goExpr(&gopast.BadExpr{})
	})
}

func TestBasic(t *testing.T) {
	test(t, `package main

import "fmt"

type a struct {
	v   map[int]chan int
	arr *[2]func()
	i   interface{}
}

var b = &a{
	arr: &[2]func(){
		nil,
		func() {},
	},
}

const c = (10 + 20) * 2

var d = b.arr[1]

var e = b.arr[:1]

var f = a.i.(func() (int))()

func foo(v ...interface{}) {}
`)
}
