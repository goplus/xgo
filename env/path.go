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
	"log"
	"os"
	"path/filepath"
	"syscall"
)

var (
	// This is set by the linker.
	defaultXGoRoot string
)

// XGOROOT returns the root of the XGo tree. It uses the XGOROOT environment variable,
// if set at process start, or else the root used during the XGo build.
func XGOROOT() string {
	xgoRoot, err := findXgoRoot()
	if err != nil {
		log.Panicln("XGOROOT not found:", err)
	}
	return xgoRoot
}

func findXgoRoot() (string, error) {
	envXgoRoot := os.Getenv(KeyXGOROOT)
	if envXgoRoot != "" {
		// XGOROOT must valid
		if isValidXgoRoot(envXgoRoot) {
			return envXgoRoot, nil
		}
		log.Panicf("\n%s (%s) is not valid\n", KeyXGOROOT, envXgoRoot)
	}

	// if parent directory is a valid xgo root, use it
	exePath, err := executableRealPath()
	if err == nil {
		dir := filepath.Dir(exePath)
		parentDir := filepath.Dir(dir)
		if parentDir != dir && isValidXgoRoot(parentDir) {
			return parentDir, nil
		}
	}

	// check defaultXGoRoot, if it is valid, use it
	if defaultXGoRoot != "" && isValidXgoRoot(defaultXGoRoot) {
		return defaultXGoRoot, nil
	}

	// Compatible with old XGOROOT
	if home := HOME(); home != "" {
		xgoRoot := filepath.Join(home, "xgo")
		if isValidXgoRoot(xgoRoot) {
			return xgoRoot, nil
		}
	}
	return "", syscall.ENOENT
}

// Mockable for testing.
var executable = func() (string, error) {
	return os.Executable()
}

func executableRealPath() (path string, err error) {
	path, err = executable()
	if err == nil {
		path, err = filepath.EvalSymlinks(path)
		if err == nil {
			path, err = filepath.Abs(path)
		}
	}
	return
}

func isFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func isValidXgoRoot(path string) bool {
	return isDirExists(filepath.Join(path, "cmd/xgo")) && isFileExists(filepath.Join(path, "go.mod"))
}
