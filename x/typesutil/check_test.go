package typesutil_test

import (
	goast "go/ast"
	"go/importer"
	goparser "go/parser"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/goplus/mod/xgomod"
	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/token"
	"github.com/goplus/xgo/x/typesutil"
	"github.com/qiniu/x/errors"
)

func init() {
	if os.Getenv("XGOROOT") == "" {
		dir, _ := os.Getwd()
		os.Setenv("XGOROOT", filepath.Clean(filepath.Join(dir, "./../..")))
	}
	typesutil.SetDebug(typesutil.DbgFlagDefault)
}

func loadFiles(fset *token.FileSet, file string, src any, goxfile string, goxsrc any, gofile string, gosrc any) ([]*ast.File, []*goast.File, error) {
	var files []*ast.File
	var gofiles []*goast.File
	if file != "" {
		f, err := parser.ParseFile(fset, file, src, 0)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, f)
	}
	if goxfile != "" {
		f, err := parser.ParseFile(fset, goxfile, goxsrc, parser.ParseGoPlusClass)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, f)
	}
	if gofile != "" {
		f, err := goparser.ParseFile(fset, gofile, gosrc, 0)
		if err != nil {
			return nil, nil, err
		}
		gofiles = append(gofiles, f)
	}
	return files, gofiles, nil
}

func checkFiles(fset *token.FileSet, file string, src any, goxfile string, goxsrc any, gofile string, gosrc any) (*typesutil.Info, *types.Info, error) {
	files, gofiles, err := loadFiles(fset, file, src, goxfile, goxsrc, gofile, gosrc)
	if err != nil {
		return nil, nil, err
	}
	return checkInfo(fset, files, gofiles, nil)
}

func checkFilesWithErrorHandler(fset *token.FileSet, file string, src any, goxfile string, goxsrc any, gofile string, gosrc any, handleErr func(error)) (*typesutil.Info, *types.Info, error) {
	files, gofiles, err := loadFiles(fset, file, src, goxfile, goxsrc, gofile, gosrc)
	if err != nil {
		return nil, nil, err
	}
	return checkInfo(fset, files, gofiles, handleErr)
}

func checkInfo(fset *token.FileSet, files []*ast.File, gofiles []*goast.File, handleErr func(error)) (*typesutil.Info, *types.Info, error) {
	conf := &types.Config{}
	conf.Importer = importer.Default()
	if handleErr == nil {
		handleErr = func(err error) {
			log.Println(err)
		}
	}
	conf.Error = handleErr
	chkOpts := &typesutil.Config{
		Types: types.NewPackage("main", "main"),
		Fset:  fset,
		Mod:   xgomod.Default,
	}
	info := &typesutil.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Overloads:  make(map[*ast.Ident]types.Object),
	}
	ginfo := &types.Info{
		Types:      make(map[goast.Expr]types.TypeAndValue),
		Defs:       make(map[*goast.Ident]types.Object),
		Uses:       make(map[*goast.Ident]types.Object),
		Implicits:  make(map[goast.Node]types.Object),
		Selections: make(map[*goast.SelectorExpr]*types.Selection),
		Scopes:     make(map[goast.Node]*types.Scope),
	}
	check := typesutil.NewChecker(conf, chkOpts, ginfo, info)
	err := check.Files(gofiles, files)
	return info, ginfo, err
}

func TestCheckFiles(t *testing.T) {
	fset := token.NewFileSet()
	info, ginfo, err := checkFiles(fset, "main.xgo", `
import "fmt"

type Point struct {
	x int
	y int
}
pt := &Point{}
pt.x = 100
pt.y = 200
fmt.Println(pt)

gopt := GoPoint{100,200}
gopt.Test()
gotest()
fmt.Println(GoValue)
fmt.Println(&Rect{100,200})
`, "Rect.gox", `
var (
	x int
	y int
)
`, "util.go", `package main
var GoValue string
type GoPoint struct {
	x int
	y int
}
func (p *GoPoint) Test() {
}
func gotest() {
}
`)
	if err != nil || info == nil || ginfo == nil {
		t.Fatalf("check failed: %v", err)
	}

	for def, obj := range info.Defs {
		o := info.ObjectOf(def)
		if o != obj {
			t.Fatal("bad obj", o)
		}
	}
	for use, obj := range info.Uses {
		o := info.ObjectOf(use)
		if o.String() != obj.String() {
			t.Fatal("bad obj", o)
		}
		typ := info.TypeOf(use)
		if typ.String() != obj.Type().String() {
			t.Fatal("bad typ", typ)
		}
	}

}

func TestCheckGoFiles(t *testing.T) {
	fset := token.NewFileSet()
	info, ginfo, err := checkFiles(fset, "", "", "", "", "main.go", `package main
type GoPoint struct {
	x int
	y int
}
func main() {
}
`)
	if err != nil || info == nil || ginfo == nil {
		t.Fatalf("check failed: %v", err)
	}
}

func TestCheckError(t *testing.T) {
	fset := token.NewFileSet()
	_, _, err := checkFiles(fset, "main.xgo", `
type Point struct {
	x int
	y int
}
pt := &Point1{}
println(pt)
`, "", "", "", "")
	if err == nil {
		t.Fatal("no error")
	}
	_, _, err = checkFiles(fset, "main.xgo", `
var i int = "hello"
`, "", "", "", "")
	if err == nil {
		t.Fatal("no error")
	}
	_, _, err = checkFiles(fset, "main.xgo", `
var nums []int
nums = append(nums, "NaN")
`, "", "", "", "")
	if err == nil {
		t.Fatal("no error")
	}
}

func TestBadFile(t *testing.T) {
	conf := &types.Config{}
	opt := &typesutil.Config{}
	opt.Fset = token.NewFileSet()
	opt.Types = types.NewPackage("", "main")
	checker := typesutil.NewChecker(conf, opt, nil, nil)
	_ = checker.Files([]*goast.File{{Name: goast.NewIdent("main")}},
		[]*ast.File{{Name: ast.NewIdent("main")}})
}

func TestCheckOverload(t *testing.T) {
	fset := token.NewFileSet()
	info, ginfo, err := checkFiles(fset, "main.xgo", `
type foo struct {
}

func (a *foo) mulInt(b int) *foo {
	return a
}

func (a *foo) mulFoo(b *foo) *foo {
	return a
}

func (foo).mul = (
	(foo).mulInt
	(foo).mulFoo
)
func addInt0() {
}

func addInt1(i int) {
}

func addInt2(i, j int) {
}

var addInt3 = func(i, j, k int) {
}

func add = (
	addInt0
	addInt1
	addInt2
	addInt3
	func(a, b string) string {
		return a + b
	}
)

var a, b *foo
var c = a.mul(100)
var d = a.mul(c)

func init() {
	add 100, 200
	add 100, 200, 300
	add("hello", "world")
}
`, "", "", "", "")
	if err != nil || info == nil || ginfo == nil {
		t.Fatalf("check failed: %v", err)
	}
	for use, ovDeclObj := range info.Overloads {
		o := info.ObjectOf(use)
		declObj, ovObjs := info.OverloadOf(use)
		if ovDeclObj != declObj {
			t.Fatal("bad overload", o)
		}
		found := false
		for _, ovObj := range ovObjs {
			if o == ovObj {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("bad overload", o)
		}
	}
	for use, o := range info.Uses {
		declObj, ovObjs := info.OverloadOf(use)
		if declObj != nil && ovObjs != nil {
			if info.Overloads[use] == nil {
				t.Fatal("bad overload", o)
			}
		}
	}
}

func TestCheckError2(t *testing.T) {
	var checkerErrs errors.List
	fset := token.NewFileSet()
	checkFilesWithErrorHandler(fset, "main.xgo", `
type GG struct {
	B bool
}

doublejump := 2
gotdoublejump := false
jumping := false
gg := &GG{B: true}
if doublejump == 2 && !gotdoublejump {
	println("Double jump activated!")
} else if doublejump == 1 && gotdoublejump {
	if !jumping {
		jumping = true
		for i := 0; i < 20; i++ {
			if gg.P {
				println("Double jump!")
			}
		}
	}
}
`, "", "", "", "", func(err error) {
		checkerErrs.Add(err)
	})

	if len(checkerErrs) > 1 {
		t.Fatal("too many errors")
	}
}
