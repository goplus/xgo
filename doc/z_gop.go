/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package doc

import (
	"go/doc"
	"strings"
)

const (
	goptPrefix = "Gopt_" // template method
	gopoPrefix = "Gopo_" // overload function/method
	gopxPrefix = "Gopx_" // type as parameters function/method
	gopPackage = "GopPackage"
	xgoPackage = "XGoPackage"
)

func isGopPackage(in *doc.Package) bool {
	for _, v := range in.Consts {
		for _, name := range v.Names {
			if name == gopPackage || name == xgoPackage {
				return true
			}
		}
	}
	return false
}

func isGopoConst(name string) bool {
	return strings.HasPrefix(name, gopoPrefix)
}

func hasGopoConst(in *doc.Value) bool {
	for _, name := range in.Names {
		if isGopoConst(name) {
			return true
		}
	}
	return false
}

func isOverload(name string) bool {
	n := len(name)
	return n > 3 && name[n-3:n-1] == "__"
}

// Func (no _ func name)
// _Func (with _ func name)
// TypeName_Method (no _ method name)
// _TypeName__Method (with _ method name)
func checkTypeMethod(name string) mthd {
	if pos := strings.IndexByte(name, '_'); pos >= 0 {
		if pos == 0 {
			t := name[1:]
			if pos = strings.Index(t, "__"); pos <= 0 {
				return mthd{"", t} // _Func
			}
			return mthd{t[:pos], t[pos+2:]} // _TypeName__Method
		}
		return mthd{name[:pos], name[pos+1:]} // TypeName_Method
	}
	return mthd{"", name} // Func
}
