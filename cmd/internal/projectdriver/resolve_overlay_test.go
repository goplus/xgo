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
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goplus/xgo/x/xgoprojs"
)

func TestResolveOverlayClassifiesDriverBeforePhysicalPreflight(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	t.Setenv("GOWORK", "off")
	root := t.TempDir()
	app := filepath.Join(root, "app")
	project := filepath.Join(app, "game")
	framework := filepath.Join(root, "framework")
	mustMkdirAll(t, project)
	mustMkdirAll(t, filepath.Join(framework, "cmd", "driver"))
	mustWriteFile(t, filepath.Join(app, "go.mod"), "module example.test/app\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(framework, "go.mod"), "module example.test/framework\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(framework, "gox.mod"), `xgo 1.8
project main.foo Game example.test/framework
driver v1 example.test/framework/cmd/driver
`)
	mustWriteFile(t, filepath.Join(framework, "cmd", "driver", "main.go"), fakeDriverSource)
	mustWriteFile(t, filepath.Join(project, "main.foo"), "// overlaid driver-backed project\n")
	overlayMod := filepath.Join(root, "overlay.mod")
	mustWriteFile(t, overlayMod, `module example.test/app

go 1.25

require example.test/framework v1.2.3 //xgo:class

replace example.test/framework => ../framework
`)
	overlayFile := filepath.Join(root, "overlay.json")
	canonicalApp, err := canonicalExistingDir(app)
	if err != nil {
		t.Fatal(err)
	}
	overlayData, err := json.Marshal(struct {
		Replace map[string]string `json:"Replace"`
	}{Replace: map[string]string{filepath.Join(canonicalApp, "go.mod"): overlayMod}})
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, overlayFile, string(overlayData))
	resolver, err := NewResolver(context.Background(), app, []string{"-overlay=" + overlayFile})
	if err != nil {
		t.Fatal(err)
	}
	canonicalProject, err := canonicalExistingDir(project)
	if err != nil {
		t.Fatal(err)
	}
	graph, graphErr := loadEffectiveGraph(context.Background(), canonicalProject, resolver.policy.graph)
	if graphErr == nil {
		matched, matchErr := overlayDriverProjectMatch(canonicalProject, graph, false)
		if matchErr != nil || !matched {
			t.Fatalf("overlay classification = %v, %v; want driver match", matched, matchErr)
		}
	} else {
		t.Fatalf("load overlay graph: %v", graphErr)
	}
	_, err = resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: project})
	if err == nil || errors.Is(err, ErrNotHandled) || !strings.Contains(err.Error(), "-overlay") {
		t.Fatalf("overlay driver Resolve() = %v, want explicit overlay rejection", err)
	}
	_, err = resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/app/game"})
	if err == nil || errors.Is(err, ErrNotHandled) || !strings.Contains(err.Error(), "-overlay") {
		t.Fatalf("overlay driver package Resolve() = %v, want explicit overlay rejection", err)
	}
}

func TestResolveOverlayLegacyTargetRemainsUnhandled(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	t.Setenv("GOWORK", "off")
	root := t.TempDir()
	app := filepath.Join(root, "app")
	mustMkdirAll(t, app)
	mustWriteFile(t, filepath.Join(app, "go.mod"), "module example.test/app\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(app, "main.go"), "package main\nfunc main() {}\n")
	overlayFile := filepath.Join(root, "overlay.json")
	overlayData, err := json.Marshal(struct {
		Replace map[string]string `json:"Replace"`
	}{Replace: map[string]string{filepath.Join(app, "main.go"): ""}})
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, overlayFile, string(overlayData))
	resolver, err := NewResolver(context.Background(), app, []string{"-overlay=" + overlayFile})
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolver.Resolve(context.Background(), &xgoprojs.DirProj{Dir: app})
	if !errors.Is(err, ErrNotHandled) {
		t.Fatalf("overlay legacy Resolve() = %v, want ErrNotHandled", err)
	}
}

func TestResolveOverlayTargetsWithSyntheticDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	t.Setenv("GOWORK", "off")
	root := t.TempDir()
	app := filepath.Join(root, "app")
	actual := filepath.Join(root, "overlay-files")
	mustMkdirAll(t, app)
	mustMkdirAll(t, actual)
	mustWriteFile(t, filepath.Join(app, "go.mod"), "module example.test/app\n\ngo 1.25\n")
	mustWriteFile(t, filepath.Join(app, "gox.mod"), `xgo 1.8
project main.foo Game example.test/app
driver v1 example.test/app/cmd/driver
`)
	mustWriteFile(t, filepath.Join(actual, "main.go"), "package game\n")
	mustWriteFile(t, filepath.Join(actual, "main.foo"), "// overlaid driver-backed project\n")
	overlayFile := filepath.Join(root, "overlay.json")
	canonicalApp, err := canonicalExistingDir(app)
	if err != nil {
		t.Fatal(err)
	}
	canonicalProject := filepath.Join(canonicalApp, "game")
	data, err := json.Marshal(struct {
		Replace map[string]string `json:"Replace"`
	}{Replace: map[string]string{
		filepath.Join(canonicalProject, "main.go"):  filepath.Join(actual, "main.go"),
		filepath.Join(canonicalProject, "main.foo"): filepath.Join(actual, "main.foo"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, overlayFile, string(data))
	resolver, err := NewResolver(context.Background(), app, []string{"-overlay=" + overlayFile})
	if err != nil {
		t.Fatal(err)
	}
	for _, arg := range []string{
		canonicalProject,
		filepath.Join(canonicalProject, "main.foo"),
		filepath.Join(canonicalProject, "..."),
	} {
		target, next, parseErr := xgoprojs.ParseOne(arg)
		if parseErr != nil || len(next) != 0 {
			t.Fatalf("ParseOne(%q) = (%T, %v, %v)", arg, target, next, parseErr)
		}
		_, resolveErr := resolver.Resolve(context.Background(), target)
		if resolveErr == nil || errors.Is(resolveErr, ErrNotHandled) || !strings.Contains(resolveErr.Error(), "-overlay") {
			t.Fatalf("overlay synthetic local target %q Resolve() = %v, want explicit overlay rejection", arg, resolveErr)
		}
	}
	_, err = resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/app/game"})
	if err == nil || errors.Is(err, ErrNotHandled) || !strings.Contains(err.Error(), "-overlay") {
		t.Fatalf("overlay synthetic package Resolve() = %v, want explicit overlay rejection", err)
	}
	_, err = resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: "example.test/app/..."})
	if err == nil || errors.Is(err, ErrNotHandled) || !strings.Contains(err.Error(), "-overlay") {
		t.Fatalf("overlay synthetic package pattern Resolve() = %v, want explicit overlay rejection", err)
	}
}
