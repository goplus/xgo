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

package optparam

// Basic optional parameters
func single(a int?) {
}

func multiple(a int, b string?, c bool?) {
}

func mixed(name string, age int?, active bool?, data []byte) {
}

// Pointer types with optional
func pointer(a *int?, b **string?) {
}

// Complex types with optional
func complex(m map[string]int?, s []int?, ch chan int?) {
}

// Array types with optional
func arrays(a [10]int?, b [5]string?) {
}

// Interface types with optional
func interfaces(r io.Reader?, w io.Writer?) {
}

// Struct types with optional
func structs(p struct{ X int }?, q struct{ Y string }?) {
}

// Function types with optional
func funcs(f func(int) string?, g func(string) error?) {
}

// Qualified identifiers with optional
func qualified(t time.Time?, d time.Duration?) {
}

// Unnamed (anonymous) parameters with optional
func unnamed(int?, string?, bool) {
}

// Mixed named and unnamed with optional
func mixedParams(a int, string?, c bool?) {
}

// All optional parameters
func allOptional(a int?, b string?, c bool?) {
}

// Optional with variadic (variadic cannot be optional)
func withVariadic(a int?, b ...string) {
}

// Type declarations with optional
type Handler func(req *Request?, resp *Response?) error

type Callback func(int?, string?) bool

// Method with optional parameters
type Server struct{}

func (s *Server) Handle(req *Request?, opts *Options?) error {
	return nil
}
