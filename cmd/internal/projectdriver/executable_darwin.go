//go:build darwin

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
	"bytes"
	"debug/macho"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

func validateHostExecutable(input *os.File, path string) error {
	file, err := macho.NewFile(input)
	if err != nil {
		return fmt.Errorf("driver output is not a Darwin executable: %w", err)
	}
	if err := file.Close(); err != nil {
		return err
	}
	currentPath, err := openFilePath(input)
	if err != nil {
		return fmt.Errorf("resolve driver output %q: %w", path, err)
	}
	identity, err := input.Stat()
	if err != nil {
		return err
	}
	if info, err := os.Lstat(currentPath); err != nil || !os.SameFile(identity, info) {
		return fmt.Errorf("driver output changed before signature validation")
	}
	cmd := exec.Command("/usr/bin/codesign", "--verify", "--strict", currentPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("driver output has no valid Darwin signature: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if info, err := os.Lstat(currentPath); err != nil || !os.SameFile(identity, info) {
		return fmt.Errorf("driver output changed during signature validation")
	}
	return nil
}

func openFilePath(file *os.File) (string, error) {
	buffer := make([]byte, unix.PathMax)
	_, _, errno := unix.Syscall(unix.SYS_FCNTL, file.Fd(), uintptr(unix.F_GETPATH), uintptr(unsafe.Pointer(&buffer[0])))
	if errno != 0 {
		return "", errno
	}
	if end := bytes.IndexByte(buffer, 0); end >= 0 {
		buffer = buffer[:end]
	}
	if len(buffer) == 0 {
		return "", fmt.Errorf("F_GETPATH returned an empty path")
	}
	return string(buffer), nil
}
