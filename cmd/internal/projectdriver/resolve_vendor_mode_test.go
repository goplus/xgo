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

package projectdriver

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/goplus/mod/modload"
	"github.com/goplus/xgo/x/xgoprojs"
)

func TestEffectiveVendorModeMatchesGoCommandDefaults(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "app")
	vendorDir := filepath.Join(app, "vendor")
	mustMkdirAll(t, vendorDir)
	goMod := filepath.Join(app, "go.mod")
	mustWriteFile(t, goMod, "module example.test/app\n\ngo 1.21\n")
	loaded, err := modload.LoadFrom(goMod, filepath.Join(app, "gox.mod"))
	if err != nil {
		t.Fatal(err)
	}

	modulePolicy := GraphPolicy{GoWork: "off"}
	if vendor, err := effectiveVendorMode(modulePolicy, goMod, loaded.File); err != nil || !vendor {
		t.Fatalf("module vendor without modules.txt = %v, %v; want true", vendor, err)
	}
	mustWriteFile(t, filepath.Join(vendorDir, "modules.txt"), "## workspace\n")
	if vendor, err := effectiveVendorMode(modulePolicy, goMod, loaded.File); err != nil || vendor {
		t.Fatalf("workspace manifest outside workspace = %v, %v; want false", vendor, err)
	}
	mustWriteFile(t, filepath.Join(vendorDir, "modules.txt"), "# module manifest\n")
	if vendor, err := effectiveVendorMode(modulePolicy, goMod, loaded.File); err != nil || !vendor {
		t.Fatalf("module manifest in module mode = %v, %v; want true", vendor, err)
	}

	goWork := filepath.Join(root, "go.work")
	mustWriteFile(t, goWork, "go 1.21\n\nuse ./app\n")
	workspaceVendor := filepath.Join(root, "vendor")
	mustMkdirAll(t, workspaceVendor)
	mustWriteFile(t, filepath.Join(workspaceVendor, "modules.txt"), "# module manifest\n")
	workspacePolicy := GraphPolicy{GoWork: goWork}
	if vendor, err := effectiveVendorMode(workspacePolicy, goMod, loaded.File); err != nil || vendor {
		t.Fatalf("module manifest in workspace mode = %v, %v; want false", vendor, err)
	}
	mustWriteFile(t, filepath.Join(workspaceVendor, "modules.txt"), "## workspace; future annotation\n")
	if vendor, err := effectiveVendorMode(workspacePolicy, goMod, loaded.File); err != nil || !vendor {
		t.Fatalf("Go 1.21 workspace manifest = %v, %v; want true", vendor, err)
	}
}

func TestResolveVendorIgnoresUnmarkedDependencyDriver(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	app := t.TempDir()
	project := filepath.Join(app, "game")
	framework := filepath.Join(app, "framework")
	mustMkdirAll(t, project)
	mustMkdirAll(t, framework)
	mustMkdirAll(t, filepath.Join(app, "vendor"))
	mustWriteFile(t, filepath.Join(app, "vendor", "modules.txt"), "# vendored\n")
	mustWriteFile(t, filepath.Join(app, "go.mod"), `module example.test/app

go 1.25

require example.test/framework v1.2.3

replace example.test/framework => ./framework
`)
	// The target module has class metadata, so vendor preflight must make a
	// positive trust decision rather than returning early before probing.
	mustWriteFile(t, filepath.Join(app, "gox.mod"), "xgo 1.8\nproject main.legacy Game example.test/app\n")
	mustWriteFile(t, filepath.Join(framework, "go.mod"), "module example.test/framework\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(framework, "gox.mod"), "xgo 1.8\nproject main.foo Game example.test/framework\ndriver v1 example.test/framework/cmd/driver\n")
	mustWriteFile(t, filepath.Join(project, "main.foo"), "// unmarked dependency driver-backed project\n")
	t.Setenv("GOWORK", "off")
	resolver, err := NewResolver(context.Background(), app, []string{"-mod=vendor"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: project})
	if !errors.Is(err, ErrNotHandled) {
		t.Fatalf("unmarked vendor dependency driver = %v, want ErrNotHandled", err)
	}
}
