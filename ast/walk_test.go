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

package ast

import (
	"slices"
	"testing"
)

func walkedNodeNames(node Node) (names []string, depth int) {
	Inspect(node, func(n Node) bool {
		if n == nil {
			depth--
			return true
		}
		depth++
		switch n := n.(type) {
		case *DomainTextLit:
			names = append(names, "DomainTextLit")
		case *MatrixLit:
			names = append(names, "MatrixLit")
		case *ElemEllipsis:
			names = append(names, "ElemEllipsis")
		case *ValueSpec:
			names = append(names, "ValueSpec")
		case *ForPhrase:
			names = append(names, "ForPhrase")
		case *ExprStmt:
			names = append(names, "ExprStmt")
		case *Ident:
			names = append(names, n.Name)
		case *BasicLit:
			names = append(names, n.Value)
		}
		return true
	})
	return names, depth
}

func TestWalkMatrixLit(t *testing.T) {
	matrixLit := &MatrixLit{
		Elts: [][]Expr{
			{
				&Ident{Name: "a"},
				&Ident{Name: "b"},
			},
			{
				&ElemEllipsis{Elt: &Ident{Name: "row"}},
			},
		},
	}

	visited, depth := walkedNodeNames(matrixLit)
	if depth != 0 {
		t.Fatalf("got traversal depth %d, want 0", depth)
	}
	if want := []string{"MatrixLit", "a", "b", "ElemEllipsis", "row"}; !slices.Equal(visited, want) {
		t.Fatalf("visited nodes = %q, want %q", visited, want)
	}
}

func TestWalkMatrixLitPrune(t *testing.T) {
	matrixLit := &MatrixLit{
		Elts: [][]Expr{
			{
				&Ident{Name: "a"},
				&ElemEllipsis{Elt: &Ident{Name: "row"}},
			},
		},
	}

	var sawMatrixLit bool
	var sawIdent bool
	Inspect(matrixLit, func(n Node) bool {
		switch n.(type) {
		case *MatrixLit:
			sawMatrixLit = true
			return false
		case *Ident:
			sawIdent = true
		}
		return true
	})

	if !sawMatrixLit {
		t.Fatal("MatrixLit was not visited")
	}
	if sawIdent {
		t.Fatal("pruned MatrixLit still visited an identifier")
	}
}

func TestWalkDomainTextLitEx(t *testing.T) {
	domainTextLit := &DomainTextLit{
		Domain: &Ident{Name: "huh"},
		Extra: &DomainTextLitEx{
			Args: []Expr{
				&Ident{Name: "ret"},
				&CallExpr{
					Fun:  &Ident{Name: "make"},
					Args: []Expr{&Ident{Name: "arg"}},
				},
			},
		},
	}

	visited, _ := walkedNodeNames(domainTextLit)
	if want := []string{"DomainTextLit", "huh", "ret", "make", "arg"}; !slices.Equal(visited, want) {
		t.Fatalf("visited nodes = %q, want %q", visited, want)
	}
}

func TestWalkValueSpecTag(t *testing.T) {
	valueSpec := &ValueSpec{
		Names: []*Ident{{Name: "field"}},
		Tag:   &BasicLit{Value: "`json:\"field\"`"},
	}

	visited, _ := walkedNodeNames(valueSpec)
	if want := []string{"ValueSpec", "field", "`json:\"field\"`"}; !slices.Equal(visited, want) {
		t.Fatalf("visited nodes = %q, want %q", visited, want)
	}
}

func TestWalkForPhraseSourceOrder(t *testing.T) {
	forPhrase := &ForPhrase{
		Key:   &Ident{Name: "k"},
		Value: &Ident{Name: "v"},
		X:     &Ident{Name: "items"},
		Init:  &ExprStmt{X: &Ident{Name: "init"}},
		Cond:  &Ident{Name: "cond"},
	}

	visited, _ := walkedNodeNames(forPhrase)
	if want := []string{"ForPhrase", "k", "v", "items", "ExprStmt", "init", "cond"}; !slices.Equal(visited, want) {
		t.Fatalf("visited nodes = %q, want %q", visited, want)
	}
}
