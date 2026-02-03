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
	"unsafe"

	"github.com/goplus/gogen"
)

// An OperandMode specifies the (addressing) mode of an operand.
type OperandMode byte

// TypeAndValue reports the type and value (for constants)
// of the corresponding expression.
type TypeAndValue struct {
	mode  OperandMode
	Type  types.Type
	Value constant.Value
}

func NewTypeAndValueForType(typ types.Type) (ret types.TypeAndValue) {
	switch t := typ.(type) {
	case *gogen.TypeType:
		typ = t.Type()
	}
	ret.Type = typ
	(*TypeAndValue)(unsafe.Pointer(&ret)).mode = TypExpr
	return
}

func NewTypeAndValueForValue(typ types.Type, val constant.Value, mode OperandMode) (ret types.TypeAndValue) {
	switch t := typ.(type) {
	case *gogen.TypeType:
		typ = t.Type()
	}
	if val != nil {
		mode = Constant
	}
	ret.Type = typ
	ret.Value = val
	(*TypeAndValue)(unsafe.Pointer(&ret)).mode = mode
	return
}

func NewTypeAndValueForCallResult(typ types.Type, val constant.Value) (ret types.TypeAndValue) {
	var mode OperandMode
	if typ == nil {
		ret.Type = &types.Tuple{}
		mode = NoValue
	} else {
		ret.Type = typ
		if val != nil {
			ret.Value = val
			mode = Constant
		} else {
			mode = Value
		}
	}
	(*TypeAndValue)(unsafe.Pointer(&ret)).mode = mode
	return
}

func NewTypeAndValueForObject(obj types.Object) (ret types.TypeAndValue) {
	var mode OperandMode
	var val constant.Value
	switch v := obj.(type) {
	case *types.Const:
		mode = Constant
		val = v.Val()
	case *types.TypeName:
		mode = TypExpr
	case *types.Var:
		mode = Variable
	case *types.Func:
		mode = Value
	case *types.Builtin:
		mode = Builtin
	case *types.Nil:
		mode = Value
	}
	ret.Type = obj.Type()
	ret.Value = val
	(*TypeAndValue)(unsafe.Pointer(&ret)).mode = mode
	return
}

func NewTypeAndValueForBuiltin(obj types.Object) (ret types.TypeAndValue) {
	ret.Type = obj.Type()
	(*TypeAndValue)(unsafe.Pointer(&ret)).mode = Builtin
	return
}
