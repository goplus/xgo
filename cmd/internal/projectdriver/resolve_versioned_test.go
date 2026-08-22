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
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/goplus/xgo/x/xgoprojs"
	"golang.org/x/mod/module"
	modzip "golang.org/x/mod/zip"
)

func TestResolveVersionedPackageUsesRequestedVersionMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	proxy := t.TempDir()
	writeModuleProxy(t, proxy, "example.test/framework", map[string]map[string]string{
		"v1.0.0": {
			"go.mod":               "module example.test/framework\n\ngo 1.25\n",
			"gox.mod":              "xgo 1.8\nproject main.foo Game example.test/framework\ndriver v1 example.test/framework/cmd/driver\n",
			"cmd/driver/driver.go": "package driver\n",
		},
	})
	writeModuleProxy(t, proxy, "example.test/app/game", map[string]map[string]string{
		"v1.3.0": {
			"go.mod":     "module example.test/app/game\n\ngo 1.25\n",
			"legacy.foo": "// nested legacy project\n",
		},
	})
	writeModuleProxy(t, proxy, "example.test/app", map[string]map[string]string{
		"v1.0.0": {
			"go.mod":  "module example.test/app\n\ngo 1.25\n",
			"main.go": "package main\nfunc main() {}\n",
		},
		"v1.1.0": {
			"go.mod":           "module example.test/app\n\ngo 1.25\n\nrequire example.test/framework v1.0.0 //xgo:class\n",
			"main.go":          "package main\nfunc main() {}\n",
			"main.foo":         "// driver-backed project\n",
			"ordinary/main.go": "package ordinary\n",
		},
		"v1.2.0": {
			"go.mod":        "module example.test/app\n\ngo 1.25\n\nrequire example.test/framework v1.0.0 //xgo:class\n",
			"main.foo":      "// XGo-only driver-backed project\n",
			"game/main.foo": "// XGo-only subdirectory driver-backed project\n",
		},
		"v1.3.0": {
			"go.mod":        "module example.test/app\n\ngo 1.25\n\nrequire example.test/framework v1.0.0 //xgo:class\n",
			"main.foo":      "// latest driver-backed project\n",
			"game/main.foo": "// parent project hidden by nested module\n",
		},
	})
	cache := t.TempDir()
	t.Cleanup(func() {
		_ = filepath.Walk(cache, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return os.Chmod(path, 0700)
			}
			return os.Chmod(path, 0600)
		})
	})
	proxyURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(proxy)}).String()
	t.Setenv("GOMODCACHE", cache)
	t.Setenv("GOPROXY", proxyURL)
	t.Setenv("GOSUMDB", "off")
	bin := t.TempDir()
	t.Setenv("GOBIN", bin)
	t.Setenv("GOWORK", "off")

	for _, test := range []struct {
		name          string
		callerVersion string
		callerClass   bool
		target        string
		wantDriver    bool
	}{
		{name: "legacy requested version ignores driver caller graph", callerVersion: "v1.1.0", callerClass: true, target: "example.test/app@v1.0.0"},
		{name: "driver requested version ignores legacy caller graph", callerVersion: "v1.0.0", target: "example.test/app@v1.1.0", wantDriver: true},
		{name: "latest query uses selected version metadata", callerVersion: "v1.0.0", target: "example.test/app@latest", wantDriver: true},
		{name: "XGo-only driver version is classified", callerVersion: "v1.0.0", target: "example.test/app@v1.2.0", wantDriver: true},
		{name: "XGo-only subdirectory uses parent module", callerVersion: "v1.0.0", target: "example.test/app/game@v1.2.0", wantDriver: true},
		{name: "nested module wins over parent prefix", callerVersion: "v1.0.0", target: "example.test/app/game@v1.3.0"},
		{name: "ordinary package in driver version remains legacy", callerVersion: "v1.0.0", target: "example.test/app/ordinary@v1.1.0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			caller := t.TempDir()
			marker := ""
			if test.callerClass {
				marker = " //xgo:class"
			}
			mustWriteFile(t, filepath.Join(caller, "go.mod"), fmt.Sprintf("module example.test/caller\n\ngo 1.25\n\nrequire example.test/app %s%s\n", test.callerVersion, marker))
			resolver, err := NewResolver(context.Background(), caller, nil)
			if err != nil {
				t.Fatal(err)
			}
			_, err = resolver.Resolve(context.Background(), &xgoprojs.PkgPathProj{Path: test.target})
			if test.wantDriver {
				if err == nil || errors.Is(err, ErrNotHandled) || !strings.Contains(err.Error(), "@version") {
					t.Fatalf("Resolve(%q) = %v, want explicit @version rejection", test.target, err)
				}
				return
			}
			if !errors.Is(err, ErrNotHandled) {
				t.Fatalf("Resolve(%q) = %v, want ErrNotHandled", test.target, err)
			}
		})
	}
	entries, err := os.ReadDir(bin)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("versioned package probe installed files into GOBIN: %v", entries)
	}
}

func TestResolveVersionedPackageCanceledContext(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the host Go command")
	}
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test/caller\n\ngo 1.25\n")
	t.Setenv("GOWORK", "off")
	bin := t.TempDir()
	t.Setenv("GOBIN", bin)
	resolver, err := NewResolver(context.Background(), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = resolver.Resolve(ctx, &xgoprojs.PkgPathProj{Path: "example.test/app@latest"})
	if !errors.Is(err, context.Canceled) || errors.Is(err, ErrNotHandled) {
		t.Fatalf("canceled versioned Resolve() = %v, want context.Canceled", err)
	}
	entries, readErr := os.ReadDir(bin)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("canceled versioned probe installed files into GOBIN: %v", entries)
	}
}

func writeModuleProxy(t *testing.T, proxy, modulePath string, versions map[string]map[string]string) {
	t.Helper()
	escaped, err := module.EscapePath(modulePath)
	if err != nil {
		t.Fatal(err)
	}
	moduleProxyDir := filepath.Join(proxy, filepath.FromSlash(escaped), "@v")
	if err := os.MkdirAll(moduleProxyDir, 0755); err != nil {
		t.Fatal(err)
	}
	versionNames := make([]string, 0, len(versions))
	for version := range versions {
		versionNames = append(versionNames, version)
	}
	sort.Strings(versionNames)
	if err := os.WriteFile(filepath.Join(moduleProxyDir, "list"), []byte(strings.Join(versionNames, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, version := range versionNames {
		files := versions[version]
		source := t.TempDir()
		for name, content := range files {
			path := filepath.Join(source, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
		}
		mod := module.Version{Path: modulePath, Version: version}
		zipPath := filepath.Join(moduleProxyDir, version+".zip")
		zipFile, err := os.Create(zipPath)
		if err != nil {
			t.Fatal(err)
		}
		zipErr := modzip.CreateFromDir(zipFile, mod, source)
		closeErr := zipFile.Close()
		if zipErr != nil {
			t.Fatal(zipErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		info := fmt.Sprintf("{\"Version\":%q,\"Time\":\"2026-01-01T00:00:00Z\"}\n", version)
		if err := os.WriteFile(filepath.Join(moduleProxyDir, version+".info"), []byte(info), 0644); err != nil {
			t.Fatal(err)
		}
		goMod, ok := files["go.mod"]
		if !ok {
			t.Fatalf("module %s@%s has no go.mod", modulePath, version)
		}
		if err := os.WriteFile(filepath.Join(moduleProxyDir, version+".mod"), []byte(goMod), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
