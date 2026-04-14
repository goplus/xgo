/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

package outline

import (
	"go/constant"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/goplus/xgo/ast"
	"github.com/goplus/xgo/parser"
	"github.com/goplus/xgo/token"
)

func TestNewPackage(t *testing.T) {
	t.Run("GoConstExpressions", func(t *testing.T) {
		pkg, conf := loadTestPackage(t, "flags", "flags.go", `package flags

type dbgFlags int

const (
	DbgFlagLoad dbgFlags = 1 << iota
	DbgFlagInstr
	DbgFlagAll = DbgFlagLoad | DbgFlagInstr
)
`)

		out, err := NewPackage("example.com/flags", pkg, conf)
		if err != nil {
			t.Fatalf("NewPackage failed: %v", err)
		}

		all, ok := out.Pkg().Scope().Lookup("DbgFlagAll").(*types.Const)
		if !ok {
			t.Fatal("DbgFlagAll constant not found in package scope")
		}
		if got, ok := constant.Int64Val(all.Val()); !ok || got != 3 {
			t.Fatalf("unexpected DbgFlagAll value: %v", all.Val())
		}
	})

	t.Run("GoGenericTypeInstantiations", func(t *testing.T) {
		pkg, conf := loadTestPackage(t, "project", "project.go", `package project

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
`)

		out, err := NewPackage("example.com/project", pkg, conf)
		if err != nil {
			t.Fatalf("NewPackage failed: %v", err)
		}

		spriteConfigObj, ok := out.Pkg().Scope().Lookup("SpriteConfig").(*types.TypeName)
		if !ok {
			t.Fatal("SpriteConfig type not found in package scope")
		}
		spriteConfig, ok := spriteConfigObj.Type().(*types.Named)
		if !ok {
			t.Fatalf("unexpected SpriteConfig type: %T", spriteConfigObj.Type())
		}
		fields, ok := spriteConfig.Underlying().(*types.Struct)
		if !ok {
			t.Fatalf("unexpected SpriteConfig underlying type: %T", spriteConfig.Underlying())
		}
		handlers, ok := fields.Field(0).Type().(*types.Named)
		if !ok || handlers.Obj().Name() != "StageItemHandlers" {
			t.Fatalf("unexpected Handlers field type: %v", fields.Field(0).Type())
		}
		if args := handlers.TypeArgs(); args.Len() != 1 || !types.Identical(args.At(0), types.NewPointer(spriteConfig)) {
			t.Fatalf("unexpected StageItemHandlers type arguments: %v", handlers.TypeArgs())
		}
	})
}

func loadTestPackage(t *testing.T, pkgName, filename, content string) (*ast.Package, *Config) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) failed: %v", path, err)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDirEx(fset, dir, parser.Config{Mode: parser.ParseComments})
	if err != nil {
		t.Fatalf("ParseDirEx(%q) failed: %v", dir, err)
	}
	pkg := pkgs[pkgName]
	if pkg == nil {
		t.Fatalf("%s package not found in %q", pkgName, dir)
	}
	return pkg, &Config{Fset: fset}
}
