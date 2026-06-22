package outline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goplus/xgo/cl"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/token"
)

func TestNewPackage(t *testing.T) {
	t.Run("GoConstExpressions", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "flags.go")
		err := os.WriteFile(path, []byte(`package flags

type dbgFlags int

const (
	DbgFlagLoad dbgFlags = 1 << iota
	DbgFlagInstr
	DbgFlagAll = DbgFlagLoad | DbgFlagInstr
)
`), 0o644)
		if err != nil {
			t.Fatalf("WriteFile(%q) failed: %v", path, err)
		}

		fset := token.NewFileSet()
		pkgs, err := parser.ParseDirEx(fset, dir, parser.Config{Mode: parser.ParseComments})
		if err != nil {
			t.Fatalf("ParseDirEx(%q) failed: %v", dir, err)
		}
		pkg := pkgs["flags"]
		if pkg == nil {
			t.Fatalf("flags package not found in %q", dir)
		}

		out, err := NewPackage("example.com/flags", pkg, &Config{Fset: fset})
		if err != nil {
			t.Fatalf("NewPackage failed: %v", err)
		}

		if out.Pkg().Scope().Lookup("DbgFlagAll") == nil {
			t.Fatal("DbgFlagAll not found in package scope")
		}
	})

	t.Run("GoGenericTypeInstantiations", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "project.go")
		err := os.WriteFile(path, []byte(`package project

type StageShape = map[string]any

type StageItemHandlers[T any] struct {
	Sprite func(StageShape) (T, error)
}

type SpriteConfig struct {
	Handlers StageItemHandlers[*SpriteConfig]
}

func AppendStageItems[T any](items []T, shape StageShape, handlers StageItemHandlers[T]) ([]T, error) {
	return items, nil
}

func (c *SpriteConfig) CloneHandlers() StageItemHandlers[*SpriteConfig] {
	return c.Handlers
}
`), 0o644)
		if err != nil {
			t.Fatalf("WriteFile(%q) failed: %v", path, err)
		}

		fset := token.NewFileSet()
		pkgs, err := parser.ParseDirEx(fset, dir, parser.Config{Mode: parser.ParseComments})
		if err != nil {
			t.Fatalf("ParseDirEx(%q) failed: %v", dir, err)
		}
		pkg := pkgs["project"]
		if pkg == nil {
			t.Fatalf("project package not found in %q", dir)
		}

		cl.SetDisableRecover(true)
		defer cl.SetDisableRecover(false)

		out, err := NewPackage("example.com/project", pkg, &Config{Fset: fset})
		if err != nil {
			t.Fatalf("NewPackage failed: %v", err)
		}

		if out.Pkg().Scope().Lookup("AppendStageItems") == nil {
			t.Fatal("AppendStageItems not found in package scope")
		}
	})
}
