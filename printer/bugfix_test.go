package printer_test

import (
	"bytes"
	"testing"

	goast "go/ast"
	goprinter "go/printer"
	gotoken "go/token"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/printer"
	"github.com/goplus/xgo/token"
)

func goIdents(name string) []*goast.Ident {
	return []*goast.Ident{{Name: name}}
}

func TestGoFormat(t *testing.T) {
	const (
		mode = goprinter.UseSpaces | goprinter.TabIndent
	)
	config := goprinter.Config{Mode: mode, Indent: 0, Tabwidth: 8}
	decl := &goast.GenDecl{Tok: gotoken.VAR, Lparen: 1, Rparen: 1}
	decl.Specs = []goast.Spec{
		&goast.ValueSpec{Names: goIdents("foo"), Type: goast.NewIdent("int")},
		&goast.ValueSpec{Names: goIdents("bar"), Type: goast.NewIdent("string")},
	}
	b := bytes.NewBuffer(nil)
	fset := gotoken.NewFileSet()
	config.Fprint(b, fset, decl)
	const codeExp = `var (
	foo int
	bar string
)`
	if code := b.String(); code != codeExp {
		t.Fatal("config.Fprint:", code, codeExp)
	}
}

func gopIdents(name string) []*ast.Ident {
	return []*ast.Ident{{Name: name}}
}

func TestXGoFormat(t *testing.T) {
	const (
		mode = printer.UseSpaces | printer.TabIndent
	)
	config := printer.Config{Mode: mode, Indent: 0, Tabwidth: 8}
	decl := &ast.GenDecl{Tok: token.VAR, Lparen: 1, Rparen: 1}
	decl.Specs = []ast.Spec{
		&ast.ValueSpec{Names: gopIdents("foo"), Type: ast.NewIdent("int")},
		&ast.ValueSpec{Names: gopIdents("bar"), Type: ast.NewIdent("string")},
	}
	b := bytes.NewBuffer(nil)
	fset := token.NewFileSet()
	config.Fprint(b, fset, decl)
	const codeExp = `var (
	foo int
	bar string
)`
	if code := b.String(); code != codeExp {
		t.Fatal("config.Fprint:", code, codeExp)
	}
}
