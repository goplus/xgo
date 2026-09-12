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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type canonicalPathTest struct {
	name string
	path string
	kind func(string) (string, error)
	want string
}

func TestCanonicalExistingPathKinds(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "dir")
	file := filepath.Join(root, "file")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	wantDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	wantFile, err := filepath.EvalSymlinks(file)
	if err != nil {
		t.Fatal(err)
	}

	tests := []canonicalPathTest{
		{name: "directory", path: dir, kind: canonicalExistingDir, want: wantDir},
		{name: "file", path: file, kind: canonicalExistingFile, want: wantFile},
	}
	if runtime.GOOS != "windows" {
		dirLink := filepath.Join(root, "dir-link")
		fileLink := filepath.Join(root, "file-link")
		if err := os.Symlink(dir, dirLink); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(file, fileLink); err != nil {
			t.Fatal(err)
		}
		tests = append(tests,
			canonicalPathTest{name: "directory symlink", path: dirLink, kind: canonicalExistingDir, want: wantDir},
			canonicalPathTest{name: "file symlink", path: fileLink, kind: canonicalExistingFile, want: wantFile},
		)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.kind(test.path)
			if err != nil || got != test.want {
				t.Fatalf("canonical path = %q, %v; want %q", got, err, test.want)
			}
		})
	}
}

func TestCanonicalExistingPathKindErrors(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	dir := filepath.Join(root, "dir")
	if err := os.WriteFile(file, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := canonicalExistingDir(file); err == nil || !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("canonicalExistingDir(file) = %v", err)
	}
	if _, err := canonicalExistingFile(dir); err == nil || !strings.Contains(err.Error(), "is not a regular non-symlink file") {
		t.Fatalf("canonicalExistingFile(dir) = %v", err)
	}
}
