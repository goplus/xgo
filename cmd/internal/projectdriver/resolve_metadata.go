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
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/goplus/mod/xgomod"
)

func declaringMetadata(origin ResolvedModule, snapshot xgomod.FileIdentity) (fileIdentity, error) {
	dir := origin.Effective().Dir
	base := filepath.Base(snapshot.Path)
	if snapshot.Path == "" || snapshot.SHA256 == "" || filepath.Dir(snapshot.Path) != dir || (base != "gox.mod" && base != "gop.mod") {
		return fileIdentity{}, fmt.Errorf("driver origin %q has an invalid declaring metadata snapshot", origin.Selected.Path)
	}
	before, err := os.Lstat(snapshot.Path)
	if err != nil {
		return fileIdentity{}, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return fileIdentity{}, fmt.Errorf("declaring metadata %q is not a regular non-symlink file", snapshot.Path)
	}
	file, err := os.Open(snapshot.Path)
	if err != nil {
		return fileIdentity{}, err
	}
	data, readErr := io.ReadAll(file)
	opened, statErr := file.Stat()
	closeErr := file.Close()
	if readErr != nil {
		return fileIdentity{}, readErr
	}
	if statErr != nil {
		return fileIdentity{}, statErr
	}
	if closeErr != nil {
		return fileIdentity{}, closeErr
	}
	after, err := os.Lstat(snapshot.Path)
	if err != nil || !os.SameFile(before, opened) || !os.SameFile(opened, after) || !after.Mode().IsRegular() {
		return fileIdentity{}, fmt.Errorf("declaring metadata %q changed after discovery", snapshot.Path)
	}
	if got := sha256Bytes(data); got != snapshot.SHA256 {
		return fileIdentity{}, fmt.Errorf("declaring metadata %q changed after discovery", snapshot.Path)
	}
	return fileIdentity(snapshot), nil
}
