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

package build_test

import (
	"bytes"
	"fmt"
	"go/printer"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/parser/fsx"
	"github.com/goplus/xgo/x/build"
)

var (
	ctx = build.Default()
)

func init() {
	cl.SetDebug(cl.FlagNoMarkAutogen)
	ctx.LoadConfig = func(cfg *cl.Config) {
		cfg.NoFileLine = true
	}
	build.RegisterClassFileType(".tspx", "MyGame", []*build.Class{
		{Ext: ".tspx", Class: "Sprite"},
	}, "github.com/goplus/xgo/cl/internal/spx")
	build.RegisterClassFileType("_yap.gox", "App", nil, "github.com/goplus/yap")
}

func gopClTest(t *testing.T, gopcode any, expected string) {
	gopClTestEx(t, "main.xgo", gopcode, expected)
}

func gopClTestEx(t *testing.T, filename string, gopcode any, expected string) {
	data, err := ctx.BuildFile(filename, gopcode)
	if err != nil {
		t.Fatalf("build gop error: %v", err)
	}
	if string(data) != expected {
		fmt.Println("build gop error:")
		fmt.Println(string(data))
		t.Fail()
	}
}

func testKind(t *testing.T, name string, proj, class bool) {
	isProj, ok := build.ClassKind(name)
	if isProj != proj || ok != class {
		t.Fatal("check classkind failed", name, isProj, ok)
	}
}

func TestKind(t *testing.T) {
	testKind(t, "Cat.gox", false, false)
	testKind(t, "Cat.spx", false, true)
	testKind(t, "main.spx", true, true)
	testKind(t, "main.gmx", true, true)
	testKind(t, "Cat.tspx", false, true)
	testKind(t, "main.tspx", true, true)
	testKind(t, "blog_yap.gox", true, true)
}

func TestGop(t *testing.T) {
	var src = `
println "XGo"
`
	var expect = `package main

import "fmt"

func main() {
	fmt.Println("XGo")
}
`
	gopClTest(t, src, expect)
	gopClTest(t, []byte(src), expect)
	gopClTest(t, bytes.NewBufferString(src), expect)
	gopClTestEx(t, `./_testdata/hello/main.xgo`, nil, expect)

	f, err := os.Open("./_testdata/hello/main.xgo")
	if err != nil {
		t.Fatal("open failed", err)
	}
	defer f.Close()
	gopClTest(t, f, expect)
}

func TestGox(t *testing.T) {
	gopClTestEx(t, "Rect.gox", `
println "XGo"
`, `package main

import "fmt"

type Rect struct {
}

func (this *Rect) Main() {
	fmt.Println("XGo")
}
func main() {
	new(Rect).Main()
}
`)
	gopClTestEx(t, "Rect.gox", `
var (
	Buffer
	v int
)
type Buffer struct {
	buf []byte
}
println "XGo"
`, `package main

import "fmt"

type Buffer struct {
	buf []byte
}
type Rect struct {
	Buffer
	v int
}

func (this *Rect) Main() {
	fmt.Println("XGo")
}
func main() {
	new(Rect).Main()
}
`)
	gopClTestEx(t, "Rect.gox", `
var (
	*Buffer
	v int
)
type Buffer struct {
	buf []byte
}
println "XGo"
`, `package main

import "fmt"

type Buffer struct {
	buf []byte
}
type Rect struct {
	*Buffer
	v int
}

func (this *Rect) Main() {
	fmt.Println("XGo")
}
func main() {
	new(Rect).Main()
}
`)
	gopClTestEx(t, "Rect.gox", `
import "bytes"
var (
	*bytes.Buffer
	v int
)
println "XGo"
`, `package main

import (
	"bytes"
	"fmt"
)

type Rect struct {
	*bytes.Buffer
	v int
}

func (this *Rect) Main() {
	fmt.Println("XGo")
}
func main() {
	new(Rect).Main()
}
`)
	gopClTestEx(t, "Rect.gox", `
import "bytes"
var (
	bytes.Buffer
	v int
)
println "XGo"
`, `package main

import (
	"bytes"
	"fmt"
)

type Rect struct {
	bytes.Buffer
	v int
}

func (this *Rect) Main() {
	fmt.Println("XGo")
}
func main() {
	new(Rect).Main()
}
`)
}

func TestBig(t *testing.T) {
	gopClTest(t, `
a := 1/2r
println a+1/2r
`, `package main

import (
	"fmt"
	"github.com/qiniu/x/xgo/ng"
	"math/big"
)

func main() {
	a := ng.Bigrat_Init__2(big.NewRat(1, 2))
	fmt.Println((ng.Bigrat).Gop_Add(a, ng.Bigrat_Init__2(big.NewRat(1, 2))))
}
`)
}

func TestIoxLines(t *testing.T) {
	gopClTest(t, `
import "io"

var r io.Reader

for line <- lines(r) {
	println line
}
`, `package main

import (
	"fmt"
	"github.com/qiniu/x/osx"
	"io"
)

var r io.Reader

func main() {
	for _gop_it := osx.Lines(r).Gop_Enum(); ; {
		var _gop_ok bool
		line, _gop_ok := _gop_it.Next()
		if !_gop_ok {
			break
		}
		fmt.Println(line)
	}
}
`)
}

func TestErrorWrap(t *testing.T) {
	gopClTest(t, `
import (
    "strconv"
)

func add(x, y string) (int, error) {
    return strconv.Atoi(x)? + strconv.Atoi(y)?, nil
}

func addSafe(x, y string) int {
    return strconv.Atoi(x)?:0 + strconv.Atoi(y)?:0
}

println add("100", "23")!

sum, err := add("10", "abc")
println sum, err

println addSafe("10", "abc")
`, `package main

import (
	"fmt"
	"github.com/qiniu/x/errors"
	"strconv"
)

func add(x string, y string) (int, error) {
	var _autoGo_1 int
	{
		var _xgo_err error
		_autoGo_1, _xgo_err = strconv.Atoi(x)
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "strconv.Atoi(x)", "main.xgo", 7, "main.add")
			return 0, _xgo_err
		}
		goto _autoGo_2
	_autoGo_2:
	}
	var _autoGo_3 int
	{
		var _xgo_err error
		_autoGo_3, _xgo_err = strconv.Atoi(y)
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "strconv.Atoi(y)", "main.xgo", 7, "main.add")
			return 0, _xgo_err
		}
		goto _autoGo_4
	_autoGo_4:
	}
	return _autoGo_1 + _autoGo_3, nil
}
func addSafe(x string, y string) int {
	return func() (_xgo_ret int) {
		var _xgo_err error
		_xgo_ret, _xgo_err = strconv.Atoi(x)
		if _xgo_err != nil {
			return 0
		}
		return
	}() + func() (_xgo_ret int) {
		var _xgo_err error
		_xgo_ret, _xgo_err = strconv.Atoi(y)
		if _xgo_err != nil {
			return 0
		}
		return
	}()
}
func main() {
	fmt.Println(func() (_xgo_ret int) {
		var _xgo_err error
		_xgo_ret, _xgo_err = add("100", "23")
		if _xgo_err != nil {
			_xgo_err = errors.NewFrame(_xgo_err, "add(\"100\", \"23\")", "main.xgo", 14, "main.main")
			panic(_xgo_err)
		}
		return
	}())
	sum, err := add("10", "abc")
	fmt.Println(sum, err)
	fmt.Println(addSafe("10", "abc"))
}
`)
}

func TestSpx(t *testing.T) {
	gopClTestEx(t, "main.tspx", `println "hi"`, `package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/spx"
)

type MyGame struct {
	spx.MyGame
}

func (this *MyGame) MainEntry() {
	fmt.Println("hi")
}
func (this *MyGame) Main() {
	spx.Gopt_MyGame_Main(this)
}
func main() {
	new(MyGame).Main()
}
`)
	gopClTestEx(t, "Cat.tspx", `println "hi"`, `package main

import (
	"fmt"
	"github.com/goplus/xgo/cl/internal/spx"
)

type Cat struct {
	spx.Sprite
	*MyGame
}
type MyGame struct {
	spx.MyGame
}

func (this *MyGame) Main() {
	spx.Gopt_MyGame_Main(this)
}
func (this *Cat) Main() {
	fmt.Println("hi")
}
func main() {
	new(MyGame).Main()
}
`)
}

func testFromDir(t *testing.T, relDir string) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal("Getwd failed:", err)
	}
	dir = path.Join(dir, relDir)
	fis, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal("ReadDir failed:", err)
	}
	for _, fi := range fis {
		name := fi.Name()
		if strings.HasPrefix(name, "_") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			testFrom(t, name, dir+"/"+name)
		})
	}
}

func testFrom(t *testing.T, name, dir string) {
	data, err := ctx.BuildDir(dir)
	if err != nil {
		t.Fatal("BuildDir failed:", err)
	}
	if chk, err := os.ReadFile(filepath.Join(dir, name+".expect")); err == nil {
		if !bytes.Equal(data, chk) {
			t.Fatalf("-- %v output check error --\n%v\n--\n%v", name, string(data), string(chk))
		}
	}
}

func TestFromTestdata(t *testing.T) {
	testFromDir(t, "./_testdata")
}

func TestFS(t *testing.T) {
	var expect = []byte(`package main

import "fmt"

func main() {
	fmt.Println("XGo")
}
`)
	data, err := ctx.BuildFSDir(fsx.Local, "./_testdata/hello")
	if err != nil {
		t.Fatal("build fs dir failed", err)
	}
	if !bytes.Equal(data, expect) {
		t.Fatal("build fs data failed", string(data))
	}
}

func TestAst(t *testing.T) {
	var expect = []byte(`package main

import "fmt"

func main() {
	fmt.Println("XGo")
}
`)
	pkg, err := ctx.ParseFSDir(fsx.Local, "./_testdata/hello")
	if err != nil {
		t.Fatal("parser fs dir failed", err)
	}
	var buf bytes.Buffer
	err = printer.Fprint(&buf, pkg.Fset, pkg.ToAst())
	if err != nil {
		t.Fatal("fprint ast error", err)
	}
	if !bytes.Equal(buf.Bytes(), expect) {
		t.Fatal("build ast data failed", buf.String())
	}
}

func TestError(t *testing.T) {
	_, err := ctx.BuildFile("main.xgo", "bad code")
	if err == nil {
		t.Fatal("BuildFile: no error?")
	}
	_, err = ctx.BuildDir("./demo/nofound")
	if err == nil {
		t.Fatal("BuildDir: no error?")
	}
	_, err = ctx.BuildFSDir(fsx.Local, "./demo/nofound")
	if err == nil {
		t.Fatal("BuildDir: no error?")
	}
	_, err = ctx.BuildFile("main.xgo", "func main()")
	if err == nil {
		t.Fatal("BuildFile: no error?")
	}
	_, err = ctx.ParseFile("main.xgo", 123)
	if err == nil {
		t.Fatal("ParseFile: no error?")
	}
	_, err = ctx.ParseFile("./demo/nofound/main.xgo", nil)
	if err == nil {
		t.Fatal("ParseFile: no error?")
	}
}

type emptyImporter struct {
}

func (i *emptyImporter) Import(path string) (*types.Package, error) {
	return nil, fmt.Errorf("not found %v", path)
}

func TestContext(t *testing.T) {
	ctx := build.NewContext(&emptyImporter{}, nil)
	_, err := ctx.BuildFile("main.xgo", `import "fmt"; fmt.Println "XGo"`)
	if err == nil {
		t.Fatal("BuildFile: no error?")
	}
}
