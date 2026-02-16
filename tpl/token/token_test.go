/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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

package token

import (
	"go/token"
	"testing"
)

func TestToken(t *testing.T) {
	if literal_beg != 3 {
		t.Fatal("literal_beg")
	}
	for i := Token(0); i < literal_end; i++ {
		s := i.String()
		if s == "RAT" || s == "UNIT" || s == "CSTRING" || s == "PYSTRING" {
			continue
		}
		if s != token.Token(i).String() {
			t.Fatal("String:", i)
		}
	}
	if IDENT.String() != "IDENT" {
		t.Fatal("IDENT")
	}
	if Token(' ').String() != "token(32)" {
		t.Fatal("token(32)")
	}
	if IDENT.Len() != 0 {
		t.Fatal("TokLen IDENT")
	}
	if Token('+').Len() != 1 {
		t.Fatal("TokLen +")
	}
	count := 0
	ForEach(0, func(tok Token, name string) int {
		count++
		return 0
	})
	if count != 28 {
		t.Fatal("ForEach:", count)
	}
	NewFileSet()
	ForEach(0, func(tok Token, name string) int {
		if tok != SHL {
			t.Fatal("ForEach:", tok)
		}
		return Break
	})
}
