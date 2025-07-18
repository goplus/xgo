/*
 * Copyright (c) 2023 The XGo Authors (xgo.dev). All rights reserved.
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

package typesutil

import (
	"go/constant"
	"go/types"
	"testing"

	"github.com/goplus/gogen"
)

func TestTypeAndValue(t *testing.T) {
	tyInt := types.Typ[types.Int]
	ty := gogen.NewTypeType(tyInt)
	ret := NewTypeAndValueForType(ty)
	if !ret.IsType() {
		t.Fatal("NewTypeAndValueForType: not type?")
	}
	ret = NewTypeAndValueForValue(tyInt, constant.MakeInt64(1), Constant)
	if ret.Value == nil {
		t.Fatal("NewTypeAndValueForValue: not const?")
	}
	ret = NewTypeAndValueForValue(ty, constant.MakeInt64(1), Constant)
	if ret.Value == nil {
		t.Fatal("NewTypeAndValueForValue: not const?")
	}
	ret = NewTypeAndValueForCallResult(tyInt, nil)
	if !ret.IsValue() {
		t.Fatal("NewTypeAndValueForCall: not value?")
	}
	ret = NewTypeAndValueForCallResult(tyInt, constant.MakeInt64(1))
	if !ret.IsValue() || ret.Value == nil {
		t.Fatal("NewTypeAndValueForCall: not const?")
	}
	ret = NewTypeAndValueForCallResult(nil, nil)
	if !ret.IsVoid() {
		t.Fatal("NewTypeAndValueForCall: not void?")
	}
	pkg := types.NewPackage("main", "main")
	ret = NewTypeAndValueForObject(types.NewConst(0, pkg, "v", tyInt, constant.MakeInt64(100)))
	if ret.Value == nil {
		t.Fatal("NewTypeAndValueForObject: not const?")
	}
	ret = NewTypeAndValueForObject(types.NewTypeName(0, pkg, "MyInt", tyInt))
	if !ret.IsType() {
		t.Fatal("NewTypeAndValueForObject: not type?")
	}
	ret = NewTypeAndValueForObject(types.NewVar(0, pkg, "v", tyInt))
	if !ret.Addressable() {
		t.Fatal("NewTypeAndValueForObject: not variable?")
	}
	ret = NewTypeAndValueForObject(types.NewFunc(0, pkg, "fn", types.NewSignature(nil, nil, nil, false)))
	if !ret.IsValue() {
		t.Fatal("NewTypeAndValueForObject: not value?")
	}
	ret = NewTypeAndValueForValue(types.Typ[types.UntypedNil], nil, Value)
	if !ret.IsNil() {
		t.Fatal("NewTypeAndValueForValue: not nil?")
	}
	ret = NewTypeAndValueForObject(types.Universe.Lookup("nil"))
	if !ret.IsNil() {
		t.Fatal("NewTypeAndValueForObject: not nil?")
	}
	ret = NewTypeAndValueForObject(types.Universe.Lookup("len"))
	if !ret.IsBuiltin() {
		t.Fatal("NewTypeAndValueForObject: not builtin?")
	}
	ret = NewTypeAndValueForBuiltin(types.Universe.Lookup("len"))
	if !ret.IsBuiltin() {
		t.Fatal("NewTypeAndValueForObject: not builtin?")
	}
}
