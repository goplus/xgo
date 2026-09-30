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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func resolveBuildOutput(cwd, requested, defaultName string) (string, error) {
	if defaultName == "" {
		return "", fmt.Errorf("empty driver executable name")
	}
	defaultName = executableName(defaultName)
	path := requested
	if path == "" {
		path = filepath.Join(cwd, defaultName)
	} else {
		trailingSeparator := strings.HasSuffix(path, string(filepath.Separator)) ||
			(runtime.GOOS == "windows" && strings.HasSuffix(path, "/"))
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, path)
		}
		info, err := os.Stat(path)
		if trailingSeparator || (err == nil && info.IsDir()) {
			path = filepath.Join(path, defaultName)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func executableName(path string) string {
	if runtime.GOOS == "windows" && !strings.EqualFold(filepath.Ext(path), ".exe") {
		return path + ".exe"
	}
	return path
}

type outputTransaction struct {
	final        string
	parentPath   string
	dir          string
	staged       string
	parent       *os.Root
	finalName    string
	workName     string
	stagedName   string
	workIdentity os.FileInfo
	keepDir      bool
	closed       bool
}

func beginOutputTransaction(final string, keepWork bool) (*outputTransaction, error) {
	parent := filepath.Dir(final)
	// Match go build -o semantics; 0o777 still honors the caller's umask.
	if err := os.MkdirAll(parent, 0o777); err != nil {
		return nil, fmt.Errorf("create driver output parent %q: %w", parent, err)
	}
	root, err := openPinnedOutputParent(parent)
	if err != nil {
		return nil, err
	}
	finalName := filepath.Base(final)
	if err := validateExistingFinal(root, finalName, final); err != nil {
		_ = root.Close()
		return nil, err
	}
	workName, err := createOutputWorkDir(root)
	if err != nil {
		_ = root.Close()
		return nil, err
	}
	workIdentity, err := root.Lstat(workName)
	if err != nil {
		_ = root.RemoveAll(workName)
		_ = root.Close()
		return nil, fmt.Errorf("inspect driver output work directory: %w", err)
	}
	stagedName := filepath.Join(workName, finalName)
	return &outputTransaction{
		final:        final,
		parentPath:   parent,
		dir:          filepath.Join(parent, workName),
		staged:       filepath.Join(parent, stagedName),
		parent:       root,
		finalName:    finalName,
		workName:     workName,
		stagedName:   stagedName,
		workIdentity: workIdentity,
		keepDir:      keepWork,
	}, nil
}

func openPinnedOutputParent(parent string) (*os.Root, error) {
	before, err := os.Stat(parent)
	if err != nil {
		return nil, fmt.Errorf("driver output parent: %w", err)
	}
	if !before.IsDir() {
		return nil, fmt.Errorf("driver output parent %q is not a directory", parent)
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return nil, fmt.Errorf("open driver output parent: %w", err)
	}
	pinned, err := root.Stat(".")
	if err != nil {
		_ = root.Close()
		return nil, fmt.Errorf("inspect pinned driver output parent: %w", err)
	}
	after, err := os.Stat(parent)
	if err != nil {
		_ = root.Close()
		return nil, fmt.Errorf("revalidate driver output parent: %w", err)
	}
	if !os.SameFile(before, pinned) || !os.SameFile(pinned, after) {
		_ = root.Close()
		return nil, fmt.Errorf("driver output parent %q changed while it was opened", parent)
	}
	return root, nil
}

func createOutputWorkDir(root *os.Root) (string, error) {
	var random [12]byte
	for range 16 {
		if _, err := rand.Read(random[:]); err != nil {
			return "", fmt.Errorf("generate driver output work directory: %w", err)
		}
		name := ".xgo-driver-output-" + hex.EncodeToString(random[:])
		if err := root.Mkdir(name, 0700); err == nil {
			return name, nil
		} else if !errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("create driver output work directory: %w", err)
		}
	}
	return "", fmt.Errorf("create driver output work directory: too many name collisions")
}

func validateExistingFinal(root *os.Root, name, displayPath string) error {
	info, err := root.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect driver output %q: %w", displayPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("driver output %q is a symlink", displayPath)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("driver output %q is not a regular file", displayPath)
	}
	return nil
}

func (tx *outputTransaction) commitContext(ctx context.Context) error {
	if tx == nil || tx.closed {
		return fmt.Errorf("driver output transaction is closed")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}
	if err := tx.checkParentPath(); err != nil {
		return err
	}
	currentInfo, err := tx.validateStagedOutput()
	if err != nil {
		return err
	}
	if err := validateExistingFinal(tx.parent, tx.finalName, tx.final); err != nil {
		return err
	}
	if finalInfo, err := tx.parent.Lstat(tx.finalName); err == nil && os.SameFile(currentInfo, finalInfo) {
		return fmt.Errorf("driver output aliases existing final output")
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("reinspect driver output %q: %w", tx.final, err)
	}
	if err := tx.checkParentPath(); err != nil {
		return err
	}
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}
	if err := tx.parent.Rename(tx.stagedName, tx.finalName); err != nil {
		state := "absent"
		if info, statErr := tx.parent.Lstat(tx.finalName); statErr == nil {
			state = info.Mode().String()
		} else if !errors.Is(statErr, fs.ErrNotExist) {
			state = statErr.Error()
		}
		return fmt.Errorf("commit driver output (final state %s): %w", state, err)
	}
	if !tx.keepDir {
		_ = tx.parent.RemoveAll(tx.workName)
	}
	syncOutputParent(tx.parent)
	tx.closed = true
	_ = tx.parent.Close()
	return nil
}

func (tx *outputTransaction) validateStagedOutput() (os.FileInfo, error) {
	workInfo, err := tx.parent.Lstat(tx.workName)
	if err != nil {
		return nil, fmt.Errorf("inspect driver output work directory: %w", err)
	}
	if workInfo.Mode()&os.ModeSymlink != 0 || !workInfo.IsDir() || !os.SameFile(workInfo, tx.workIdentity) {
		return nil, fmt.Errorf("driver output work directory changed during driver execution")
	}
	info, err := tx.parent.Lstat(tx.stagedName)
	if err != nil {
		return nil, fmt.Errorf("driver output: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("driver output %q is not a regular non-symlink file", tx.staged)
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("driver output %q is empty", tx.staged)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
		return nil, fmt.Errorf("driver output %q is not executable", tx.staged)
	}
	file, err := tx.parent.Open(tx.stagedName)
	if err != nil {
		return nil, fmt.Errorf("open driver output: %w", err)
	}
	openedInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("inspect open driver output: %w", err)
	}
	if !os.SameFile(info, openedInfo) || !openedInfo.Mode().IsRegular() {
		_ = file.Close()
		return nil, fmt.Errorf("driver output changed while it was opened")
	}
	// Windows and AIX cannot flush this read-only validation handle.
	if runtime.GOOS != "windows" && runtime.GOOS != "aix" {
		if err := file.Sync(); err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("sync driver output: %w", err)
		}
	}
	if err := validateHostExecutable(file, tx.staged); err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close driver output: %w", err)
	}
	currentInfo, err := tx.parent.Lstat(tx.stagedName)
	if err != nil || !os.SameFile(openedInfo, currentInfo) {
		return nil, fmt.Errorf("driver output changed after validation")
	}
	return currentInfo, nil
}

func (tx *outputTransaction) checkParentPath() error {
	pinned, err := tx.parent.Stat(".")
	if err != nil {
		return fmt.Errorf("inspect pinned driver output parent: %w", err)
	}
	current, err := os.Stat(tx.parentPath)
	if err != nil {
		return fmt.Errorf("revalidate driver output parent %q: %w", tx.parentPath, err)
	}
	if !current.IsDir() || !os.SameFile(pinned, current) {
		return fmt.Errorf("driver output parent %q changed during driver execution", tx.parentPath)
	}
	return nil
}

func syncOutputParent(root *os.Root) {
	dir, err := root.Open(".")
	if err != nil {
		return
	}
	_ = dir.Sync()
	_ = dir.Close()
}

func (tx *outputTransaction) abort() {
	if tx == nil || tx.closed {
		return
	}
	if !tx.keepDir {
		if info, err := tx.parent.Lstat(tx.workName); err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 && os.SameFile(info, tx.workIdentity) {
			_ = tx.parent.RemoveAll(tx.workName)
		}
	}
	tx.closed = true
	_ = tx.parent.Close()
}
