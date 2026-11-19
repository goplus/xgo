package cl

import (
	"go/types"
	"testing"

	"github.com/goplus/gogen"
	"github.com/goplus/xgo/ast"
)

func TestToStructTypeEmbeddedAny(t *testing.T) {
	for _, tt := range []struct {
		name    string
		defined bool
	}{
		{name: "Predeclared"},
		{name: "Defined", defined: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			conf := &gogen.Config{
				NewBuiltin: func(pkg *gogen.Package, _ *gogen.Config) *types.Package {
					builtin := types.NewPackage("", "")
					var ref gogen.PkgRef
					initBuiltin(pkg, builtin, ref, ref, ref, ref, ref, ref)
					return builtin
				},
			}
			pkg := gogen.NewPackage("", "foo", conf)
			ctx := &blockCtx{pkg: pkg, cb: pkg.CB(), pkgCtx: &pkgCtx{}}
			id := ast.NewIdent("any")
			node := &ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{{Type: id}}}}
			want := gogen.TyEmptyInterface
			if tt.defined {
				obj := types.NewTypeName(0, pkg.Types, "any", nil)
				want = types.NewNamed(obj, types.Typ[types.Int], nil)
				pkg.Types.Scope().Insert(obj)
			}
			typ := toStructType(ctx, node)
			if typ.NumFields() != 1 {
				t.Fatalf("field count = %d, want 1", typ.NumFields())
			}
			field := typ.Field(0)
			if field.Name() != "any" || field.Type() != want || !field.Embedded() {
				t.Fatalf("field = %v, want embedded any of type %v", field, want)
			}
		})
	}
}
