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

package format

import (
	"testing"
)

func testFormat(t *testing.T, name string, src, expect string) {
	t.Run(name, func(t *testing.T) {
		result, err := XGostyleSource([]byte(src), name)
		if err != nil {
			t.Fatal("format.Source failed:", err)
		}
		if ret := string(result); ret != expect {
			t.Fatalf("%s => Expect:\n%s\n=> Got:\n%s\n", name, expect, ret)
		}
	})
}

// -----------------------------------------------------------------------------

func TestMain(t *testing.T) {
	testFormat(t, "hello world 1", `package main

import "fmt"

// this is main
func main() {
	// say hello
	fmt.Println("Hello world")
}
`, `// this is main

// say hello
echo "Hello world"
`)
	testFormat(t, "hello world 2", `package main

import "fmt"

// this is main
func main() {
	// say hello
	fmt.Println("Hello world")
}

func f() {
}
`, `// this is main
func main() {
	// say hello
	echo "Hello world"
}

func f() {
}
`)
	testFormat(t, "hello world 3", `package main

import "fmt"

func f() {
}
`, `func f() {
}
`)
}

// -----------------------------------------------------------------------------

func TestPrint(t *testing.T) {
	testFormat(t, "print", `package main

import "fmt"

func f() {
	fmt.Print("hello")
}
`, `func f() {
	print "hello"
}
`)
}

func TestPrintf(t *testing.T) {
	testFormat(t, "print", `package main

import "fmt"

func f() {
	fmt.Printf("hello")
}
`, `func f() {
	printf "hello"
}
`)
}

func TestPrintln(t *testing.T) {
	testFormat(t, "print", `package main

import "fmt"

func f() {
	fmt.Println("hello")
}
`, `func f() {
	echo "hello"
}
`)
}

func TestPrintlnGroup(t *testing.T) {
	testFormat(t, "print", `package main

import (
	"fmt"
)

func f() {
	fmt.Println("hello")
}
`, `func f() {
	echo "hello"
}
`)
}

func TestPrintlnWithOtherFmtCalls(t *testing.T) {
	testFormat(t, "print", `package main

import . "errors"
import "fmt"

func f() {
	fmt.Errorf("%w", New("hello"))
	fmt.Println("hello")
}
`, `import . "errors"

func f() {
	errorf "%w", New("hello")
	echo "hello"
}
`)
}

func TestPrintlnWithOtherFmtCallsGroup(t *testing.T) {
	testFormat(t, "print", `package main

import (
	"errors"
	"fmt"
)

func f() {
	fmt.Errorf("%w", errors.New("hello"))
	fmt.println "hello"
}
`, `import (
	"errors"
)

func f() {
	errorf "%w", errors.new("hello")
	echo "hello"
}
`)
}

func TestPrintlnWithOtherFmtCallsWithAssign(t *testing.T) {
	testFormat(t, "print", `package main

import "errors"
import "fmt"

func f() {
	_ = fmt.Errorf("%w", errors.New("hello"))
	fmt.println("hello")
}
`, `import "errors"

func f() {
	_ = errorf("%w", errors.new("hello"))
	echo "hello"
}
`)
}

func TestPrintlnWithOtherFmtCallsWithGroupWithAssign(t *testing.T) {
	testFormat(t, "print", `package main

import (
	"errors"
	"fmt"
)

func f() {
	_ = fmt.Errorf("%w", errors.New("hello"))
	fmt.Println("hello")
}
`, `import (
	"errors"
)

func f() {
	_ = errorf("%w", errors.new("hello"))
	echo "hello"
}
`)
}

func TestPrintlnWithOtherFmtDecls(t *testing.T) {
	testFormat(t, "print", `package main

import "fmt"

func f() {
	_ = fmt.Stringer
	fmt.Println("hello")
}
`, `import "fmt"

func f() {
	_ = fmt.Stringer
	echo "hello"
}
`)
}

func TestPrintlnWithOtherFmtVars(t *testing.T) {
	testFormat(t, "print", `package main

import "fmt"

func f() {
	var _ fmt.Stringer
	fmt.Println("hello")
}
`, `import "fmt"

func f() {
	var _ fmt.Stringer
	echo "hello"
}
`)
}

func TestPrintlnWithOtherFmtType(t *testing.T) {
	testFormat(t, "print", `package main

import "fmt"

func f() {
	var _ struct {
		fmt.Stringer
		fn func()
	}
	fmt.Println("hello")
}
`, `import "fmt"

func f() {
	var _ struct {
		fmt.Stringer
		fn func()
	}
	echo "hello"
}
`)
}

func TestPrintlnImportAlias(t *testing.T) {
	testFormat(t, "print", `package main

import fmt1 "fmt"

func f() {
	fmt1.Println("hello")
}
`, `func f() {
	echo "hello"
}
`)
}

func TestPrintlnImportMultiAliases(t *testing.T) {
	testFormat(t, "print", `package main

import (
	fmt1 "fmt"
	fmt2 "fmt"
)

func f() {
	fmt1.Println(1)
	fmt2.Println(2)
}
`, `func f() {
	echo 1
	echo 2
}
`)
}

func TestPrintlnImportMultiAliasesDifferentGroups(t *testing.T) {
	testFormat(t, "print", `package main

import "fmt"

import (
	fmt1 "fmt"
	fmt2 "fmt"
)

func f() {
	var _ fmt.Stringer
	fmt1.Println(1)
	fmt2.Println(2)
}
`, `import "fmt"

func f() {
	var _ fmt.Stringer
	echo 1
	echo 2
}
`)
}

func TestErrorfWithPackageLevelVar(t *testing.T) {
	testFormat(t, "print", `package main

import "errors"
import "fmt"

var _ = fmt.Errorf("hello %w", errors.New("world"))
`, `import "errors"

var _ = errorf("hello %w", errors.new("world"))
`)
}

func TestPrintlnWithFmtVarNoImport(t *testing.T) {
	testFormat(t, "print", `package main

func f() {
	var fmt Foo
	fmt.Println(1)
}
`, `func f() {
	var fmt Foo
	fmt.println 1
}
`)
}

func TestPrintlnWithFmtVar(t *testing.T) {
	testFormat(t, "print", `package main

import "fmt"

var _ fmt.Stringer

func f() {
	var fmt Foo
	fmt.Println(1)
}
`, `import "fmt"

var _ fmt.Stringer

func f() {
	var fmt Foo
	fmt.println 1
}
`)
}

// todo: fix inner scope vars
// func TestPrintlnWithScopedFmtVar(t *testing.T) {
// 	testFormat(t, "print", `package main
//
// import "fmt"
//
// func f() {
// 	{
// 		var fmt Foo
// 		_ = fmt
// 	}
// 	fmt.Println(1)
// }
// `, `func f() {
// 	{
// 		var fmt Foo
// 		_ = fmt
// 	}
// 	println 1
// }
// `)
// }

func TestPrintlnWithFmtVarAfter(t *testing.T) {
	testFormat(t, "print", `package main

import "fmt"

func f() {
	fmt.Println(1)
	var fmt Foo
	_ = fmt
}
`, `func f() {
	echo 1
	var fmt Foo
	_ = fmt
}
`)
}

func TestPrintlnWithPackageLevelFmtVar(t *testing.T) {
	testFormat(t, "print", `package main

var fmt Foo

func f() {
	fmt.Println(1)
}
`, `var fmt Foo

func f() {
	fmt.println 1
}
`)
}

func TestPrintlnWithPackageLevelFmtVarAfter(t *testing.T) {
	testFormat(t, "print", `package main

func f() {
	fmt.Println(1)
}

var fmt Foo
`, `func f() {
	fmt.println 1
}

var fmt Foo
`)
}

func TestPrintlnWithVarFromCall(t *testing.T) {
	testFormat(t, "print", `package main

func f() {
	var fmt = Foo()
	fmt.Println(1)
}
`, `func f() {
	var fmt = Foo()
	fmt.println 1
}
`)
}

func TestLambdaFromFuncLit(t *testing.T) {
	testFormat(t, "funclit to lambda", `package main
println(demo(func(n int) int {
	return n+100
}))
println(demo(func(n int) int {
	return n+100
}),200)
demo1(100, func(n int) {
	println(n)
})
demo1(100, func(int) {
	println(100)
})
demo2(300, func(n1, n2 int) int {
	return(n1 + n2)
})
demo2(300, func(int, int) int {
	return -600
})
demo3(100,func(n1,n2 int)(int) {
	return n1+n2,n1-n2
},100)
demo3(100,func(n1,n2 int)(v int) {
	return n1+n2,n1-n2
},100)
demo4(100,func(n1,n2 int)(a,b int) {
	println(a,b)
	return n1+n2,n1-n2
},100)
`, `println demo(n => n + 100)
println demo(n => n + 100), 200
demo1 100, n => {
	println n
}
demo1 100, _ => {
	println 100
}
demo2 300, (n1, n2) => (n1 + n2)

demo2 300, (_, _) => -600

demo3 100, (n1, n2) => {
	return n1 + n2, n1 - n2
}, 100
demo3 100, func(n1, n2 int) (v int) {
	return n1 + n2, n1 - n2
}, 100
demo4 100, func(n1, n2 int) (a, b int) {
	println a, b
	return n1 + n2, n1 - n2
}, 100
`)
}
