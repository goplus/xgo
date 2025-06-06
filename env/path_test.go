/*
 * Copyright (c) 2021 The XGo Authors (xgo.dev). All rights reserved.
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

package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goplus/mod"
)

func findGoModFile(dir string) (modfile string, noCacheFile bool, err error) {
	modfile, err = mod.GOMOD(dir)
	if err != nil {
		xgoRoot, err := findXgoRoot()
		if err == nil {
			modfile = filepath.Join(xgoRoot, "go.mod")
			return modfile, true, nil
		}
	}
	return
}

// Common testing directory structure:
// testing_root/
//
//	  src/
//		   subdir/
//	  valid_xgoroot/
//		   go.mod
//		   go.sum
//	    cmd/xgo/
func makeTestDir(t *testing.T) (root string, src string, xgoRoot string) {
	root, _ = filepath.EvalSymlinks(t.TempDir())
	src = filepath.Join(root, "src")
	xgoRoot = filepath.Join(root, "valid_xgoroot")
	makeValidXgoRoot(xgoRoot)
	os.Mkdir(src, 0755)
	return
}

func makeValidXgoRoot(root string) {
	os.Mkdir(root, 0755)
	os.MkdirAll(filepath.Join(root, "cmd/xgo"), 0755)
	os.WriteFile(filepath.Join(root, "go.mod"), nil, 0644)
	os.WriteFile(filepath.Join(root, "go.sum"), nil, 0644)
}

func writeDummyFile(path string) {
	os.WriteFile(path, nil, 0644)
}

func cleanup() {
	os.Setenv("XGOROOT", "")
	os.Setenv(envHOME, "")
	defaultXGoRoot = ""
}

func TestBasic(t *testing.T) {
	defaultXGoRoot = ".."
	if XGOROOT() == "" {
		t.Fatal("TestBasic failed")
	}
	defaultXGoRoot = ""
}

func TestFindGoModFileInGoModDir(t *testing.T) {
	cleanup()

	t.Run("the src/ is a valid mod dir", func(tt *testing.T) {
		tt.Cleanup(cleanup)
		_, src, _ := makeTestDir(tt)
		subdir := filepath.Join(src, "subdir")
		writeDummyFile(filepath.Join(src, "go.mod"))
		os.Mkdir(subdir, 0755)

		{
			modfile, noCacheFile, err := findGoModFile(src)

			if err != nil || modfile != filepath.Join(src, "go.mod") || noCacheFile {
				tt.Fatal("got:", modfile, noCacheFile, err)
			}
		}

		{
			// Should found go.mod in parent dir
			modfile, noCacheFile, err := findGoModFile(subdir)

			if err != nil || modfile != filepath.Join(src, "go.mod") || noCacheFile {
				tt.Fatal("got:", modfile, noCacheFile, err)
			}
		}
	})

	t.Run("the src/ is not a valid mod dir", func(tt *testing.T) {
		tt.Cleanup(cleanup)
		_, src, _ := makeTestDir(tt)

		modfile, noCacheFile, err := findGoModFile(src)

		if err == nil {
			tt.Fatal("should not found the mod file, but got:", modfile, noCacheFile)
		}
	})
}

func TestFindGoModFileInGopRoot(t *testing.T) {
	originDir, _ := os.Getwd()
	origiExecutable := executable
	home := filepath.Join(os.TempDir(), "test_home")
	os.Mkdir(home, 0755)
	t.Cleanup(func() {
		os.Chdir(originDir)
		os.RemoveAll(home)
		executable = origiExecutable
	})

	bin := filepath.Join(home, "bin")

	// Don't find go.mod in gop source dir when testing
	os.Chdir(home)

	cleanupAll := func() {
		cleanup()
		executable = func() (string, error) {
			return filepath.Join(bin, "run"), nil
		}
	}
	cleanupAll()

	t.Run("without xgo root", func(tt *testing.T) {
		tt.Cleanup(cleanupAll)
		root, _, _ := makeTestDir(tt)

		modfile, noCacheFile, err := findGoModFile(root)

		if err == nil || noCacheFile || modfile != "" {
			tt.Fatal("should not found go.mod without xgo root, got:", modfile, noCacheFile, err)
		}
	})

	t.Run("set XGOROOT to a valid xgoroot path", func(tt *testing.T) {
		tt.Cleanup(cleanupAll)
		_, src, xgoRoot := makeTestDir(tt)

		os.Setenv("XGOROOT", xgoRoot)
		modfile, noCacheFile, err := findGoModFile(src)

		if err != nil || modfile != filepath.Join(xgoRoot, "go.mod") || !noCacheFile {
			tt.Fatal("should found mod file in XGOROOT, got:", modfile, noCacheFile, err)
		}
	})

	t.Run("set XGOROOT to an invalid xgoroot path", func(tt *testing.T) {
		tt.Cleanup(cleanupAll)
		root, src, _ := makeTestDir(tt)
		invalidXgoRoot := filepath.Join(root, "invalid_xgoroot")

		defer func() {
			r := recover()
			if r == nil {
				tt.Fatal("should panic, but not")
			}
		}()

		os.Setenv("XGOROOT", invalidXgoRoot)
		findGoModFile(src)
	})

	t.Run("set defaultXGoRoot to a valid xgoroot path", func(tt *testing.T) {
		tt.Cleanup(cleanupAll)
		_, src, xgoRoot := makeTestDir(tt)

		defaultXGoRoot = xgoRoot
		modfile, noCacheFile, err := findGoModFile(src)

		if err != nil || modfile != filepath.Join(xgoRoot, "go.mod") || !noCacheFile {
			tt.Fatal("should found go.mod in the dir of defaultXGoRoot, got:", modfile, noCacheFile, err)
		}
	})

	t.Run("set defaultXGoRoot to an invalid path", func(tt *testing.T) {
		tt.Cleanup(cleanupAll)
		root, src, _ := makeTestDir(tt)
		invalidXgoRoot := filepath.Join(root, "invalid_xgoroot")

		defaultXGoRoot = invalidXgoRoot
		{
			modfile, noCacheFile, err := findGoModFile(src)

			if err == nil || noCacheFile || modfile != "" {
				tt.Fatal("should not found go.mod when defaultXGoRoot isn't exists, got:", modfile, noCacheFile, err)
			}
		}

		{
			os.Mkdir(invalidXgoRoot, 0755)
			modfile, noCacheFile, err := findGoModFile(src)

			if err == nil || noCacheFile || modfile != "" {
				tt.Fatal("should not found go.mod when defaultXGoRoot isn't an valid gop root dir, got:", modfile, noCacheFile, err)
			}
		}
	})

	t.Run("use $HOME/xgo", func(tt *testing.T) {
		tt.Cleanup(cleanupAll)
		root, src, _ := makeTestDir(tt)
		home := filepath.Join(root, "home")
		os.Mkdir(home, 0755)
		os.Setenv(envHOME, home)

		{
			xgoRoot := filepath.Join(home, "xgo")
			makeValidXgoRoot(xgoRoot)

			modfile, noCacheFile, err := findGoModFile(src)

			if err != nil || !noCacheFile || modfile != filepath.Join(xgoRoot, "go.mod") {
				tt.Fatal("should found go.mod in $HOME/gop, but got:", modfile, noCacheFile, err)
			}
		}
	})

	t.Run("check if parent dir of the executable is valid gop root", func(tt *testing.T) {
		tt.Cleanup(cleanupAll)
		_, src, xgoRoot := makeTestDir(tt)
		bin := filepath.Join(xgoRoot, "bin")
		exePath := filepath.Join(bin, "run")
		os.Mkdir(bin, 0755)
		writeDummyFile(exePath)

		// Mock executable location
		executable = func() (string, error) {
			return exePath, nil
		}

		modfile, noCacheFile, err := findGoModFile(src)

		if err != nil || !noCacheFile || modfile != filepath.Join(xgoRoot, "go.mod") {
			tt.Fatal("should found go.mod in xgoRoot, but got:", modfile, noCacheFile, err)
		}
	})

	t.Run("test xgo root priority", func(tt *testing.T) {
		tt.Cleanup(cleanupAll)
		root, src, _ := makeTestDir(tt)

		tt.Run("without xgo root", func(tt *testing.T) {
			modfile, noCacheFile, err := findGoModFile(src)

			if err == nil || noCacheFile || modfile != "" {
				tt.Fatal("should not found go.mod without xgo root, got:", modfile, noCacheFile, err)
			}
		})

		tt.Run("set defaultXGoRoot to a valid xgo root dir", func(tt *testing.T) {
			newXgoRoot := filepath.Join(root, "new_xgo_root")
			makeValidXgoRoot(newXgoRoot)

			defaultXGoRoot = newXgoRoot
			modfile, noCacheFile, err := findGoModFile(src)

			if err != nil || !noCacheFile || modfile != filepath.Join(newXgoRoot, "go.mod") {
				tt.Fatal("should found go.mod in new_xgo_root/, but got:", modfile, noCacheFile, err)
			}
		})

		tt.Run("the executable's parent dir is a valid xgo root dir", func(tt *testing.T) {
			newGopRoot2 := filepath.Join(root, "new_xgo_root2")
			makeValidXgoRoot(newGopRoot2)
			bin := filepath.Join(newGopRoot2, "bin")
			exePath := filepath.Join(bin, "run")
			os.Mkdir(bin, 0755)
			writeDummyFile(exePath)
			// Mock executable location
			executable = func() (string, error) {
				return exePath, nil
			}

			modfile, noCacheFile, err := findGoModFile(src)

			if err != nil || !noCacheFile || modfile != filepath.Join(newGopRoot2, "go.mod") {
				tt.Fatal("should found go.mod in new_xgo_root2/, but got:", modfile, noCacheFile, err)
			}
		})

		tt.Run("set XGOROOT to an invalid xgo root dir", func(tt *testing.T) {
			newGopRoot3 := filepath.Join(root, "new_xgo_root3")

			defer func() {
				r := recover()
				if r == nil {
					tt.Fatal("should panic, but not")
				}
			}()

			os.Setenv("XGOROOT", newGopRoot3)
			findGoModFile(src)
		})

		tt.Run("set XGOROOT to a valid xgo root dir", func(tt *testing.T) {
			newGopRoot4 := filepath.Join(root, "new_xgo_root4")
			makeValidXgoRoot(newGopRoot4)

			os.Setenv("XGOROOT", newGopRoot4)
			modfile, noCacheFile, err := findGoModFile(src)

			if err != nil || !noCacheFile || modfile != filepath.Join(newGopRoot4, "go.mod") {
				tt.Fatal("should found go.mod in new_xgo_root3/, but got:", modfile, noCacheFile, err)
			}
		})
	})
}
