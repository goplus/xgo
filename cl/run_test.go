//go:build !genjs

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

package cl_test

import (
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/cl/cltest"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/parser/fsx/memfs"
	"github.com/goplus/xgo/scanner"
)

var (
	tmpDir     string
	tmpFileIdx int64
)

func init() {
	home, err := os.Getwd()
	check(err)

	tmpDir = home + "/.xgo/tmp/"
	err = os.MkdirAll(tmpDir, 0755)
	check(err)
}

func check(err error) {
	if err != nil {
		log.Panicln(err)
	}
}

func checkWith(err error, stdout, stderr io.Writer) {
	if err != nil {
		fatalWith(err, stdout, stderr)
	}
}

func fatalWith(err error, stdout, stderr io.Writer) {
	if o, ok := getBytes(stdout, stderr); ok {
		os.Stderr.Write(o.Bytes())
	}
	log.Panicln(err)
}

type iBytes interface {
	Bytes() []byte
}

func getBytes(stdout, stderr io.Writer) (o iBytes, ok bool) {
	if o, ok = stderr.(iBytes); ok {
		return
	}
	o, ok = stdout.(iBytes)
	return
}

func goRun(_ *testing.T, code []byte) string {
	idx := atomic.AddInt64(&tmpFileIdx, 1)
	infile := tmpDir + strconv.FormatInt(idx, 10) + ".go"
	err := os.WriteFile(infile, []byte(code), 0666)
	check(err)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", infile)
	cmd.Dir = tmpDir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	os.Remove(infile)
	checkWith(err, &stdout, &stderr)
	return stdout.String()
}

func genGo(t *testing.T, conf *cl.Config, gopcode string) []byte {
	cl.SetDisableRecover(true)
	defer cl.SetDisableRecover(false)

	fs := memfs.SingleFile("/foo", "bar.xgo", gopcode)
	pkgs, err := parser.ParseFSDir(cltest.Conf.Fset, fs, "/foo", parser.Config{Mode: parser.ParseComments})
	if err != nil {
		scanner.PrintError(os.Stderr, err)
		t.Fatal("ParseFSDir:", err)
	}
	pkg, err := cl.NewPackage("", pkgs["main"], conf)
	if err != nil {
		t.Fatal("NewPackage:", err)
	}
	var b bytes.Buffer
	err = pkg.WriteTo(&b)
	if err != nil {
		t.Fatal("gogen.WriteTo failed:", err)
	}
	return b.Bytes()
}

func testRun(t *testing.T, gopcode, expected string) {
	code := genGo(t, cltest.Conf, gopcode)
	result := goRun(t, code)
	if result != expected {
		t.Fatalf("=> Result:\n%s\n=> Expected:\n%s\n", result, expected)
	}
}

func testRunType(t *testing.T, typ, gopcodeT, expected string) {
	gopcode := strings.ReplaceAll(gopcodeT, "$(type)", typ)
	testRun(t, gopcode, expected)
}

// -----------------------------------------------------------------------------

const (
	testType_inc_code, testType_inc_ret = `
{
	var x $(type) = 1
	var y = +x
	x++
	x+=10
	println x, y
}
`, `12 1
`
	testType_dec_code, testType_dec_ret = `
{
	var x $(type) = 0
	x--
	println x
}
`, `-1
`
	testType_init_code, testType_init_ret = `
{
	var x $(type) = 1 << 65
	var y = x >> 63
	var z = x >> 65
	println x
	println y, z
}
`, `36893488147419103232
4 1
`
	testType_twoval_code, testType_twoval_ret = `
{
	var x $(type) = 1 << 65
	var y = (x >> 2) - 1
	v1, ok1 := int64(x)
	v2, ok2 := int64(y)
	println v1, ok1
	println v2, ok2
}
`, `0 false
9223372036854775807 true
`
	testType_cast_code, testType_cast_ret = `
{
	println $(type)(1 << 65), $(type)()
}
`, `36893488147419103232 0
`
	testType_printf_code, testType_printf_ret = `
{
	var x $(type) = 1
	printf "%4d\n", x
}
`, `   1
`
	testTypescanf_code, testType_scanf_ret = `
import "fmt"
{
	var name string
	var age $(type)
	fmt.Sscanf("Kim is 22 years old", "%s is %d years old", &name, &age)
	println name, age
}
`, `Kim 22
`
)

const (
	testType_com_code, testType_com_ret = testType_inc_code + testType_init_code + testType_cast_code + testType_printf_code + testType_twoval_code,
		testType_inc_ret + testType_init_ret + testType_cast_ret + testType_printf_ret + testType_twoval_ret
	testType_fixint_code, testType_fixint_ret = testTypescanf_code + testType_com_code,
		testType_scanf_ret + testType_com_ret
)

func TestUint128_run(t *testing.T) {
	testRunType(t, "uint128", testType_fixint_code, testType_fixint_ret)
}

func TestInt128_run(t *testing.T) {
	testRunType(t, "int128", testType_fixint_code+testType_dec_code, testType_fixint_ret+testType_dec_ret)
}

func TestBigint_run(t *testing.T) {
	testRunType(t, "bigint", testType_com_code+testType_dec_code, testType_com_ret+testType_dec_ret)
}

// -----------------------------------------------------------------------------
