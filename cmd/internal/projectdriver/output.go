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
		if info, err := os.Stat(path); err == nil && info.IsDir() || trailingSeparator {
			path = filepath.Join(path, defaultName)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return executableName(filepath.Clean(abs)), nil
}

func executableName(path string) string {
	if runtime.GOOS == "windows" && !strings.EqualFold(filepath.Ext(path), ".exe") {
		return path + ".exe"
	}
	return path
}

func beginOutputTransaction(final string, keepWork bool) (*outputTransaction, error) {
	parent := filepath.Dir(final)
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

func (tx *outputTransaction) abort() {
	if tx == nil || tx.closed {
		return
	}
	if !tx.keepDir {
		_ = tx.parent.RemoveAll(tx.workName)
	}
	tx.closed = true
	_ = tx.parent.Close()
}

// commitContext validates and publishes staged output.
// Cancellation is checked immediately before the commit rename.
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
	workInfo, err := tx.parent.Lstat(tx.workName)
	if err != nil {
		return fmt.Errorf("inspect driver output work directory: %w", err)
	}
	if workInfo.Mode()&os.ModeSymlink != 0 || !workInfo.IsDir() || !os.SameFile(workInfo, tx.workIdentity) {
		return fmt.Errorf("driver output work directory changed during driver execution")
	}
	entries, err := fs.ReadDir(tx.parent.FS(), tx.workName)
	if err != nil {
		return err
	}
	if len(entries) != 1 || entries[0].Name() != tx.finalName {
		return fmt.Errorf("driver must create exactly one staged output")
	}
	info, err := tx.parent.Lstat(tx.stagedName)
	if err != nil {
		return fmt.Errorf("driver output: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("driver output %q is not a regular non-symlink file", tx.staged)
	}
	if info.Size() == 0 {
		return fmt.Errorf("driver output %q is empty", tx.staged)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("driver output %q is not executable", tx.staged)
	}
	file, err := tx.parent.Open(tx.stagedName)
	if err != nil {
		return fmt.Errorf("open driver output: %w", err)
	}
	openedInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("inspect open driver output: %w", err)
	}
	if !os.SameFile(info, openedInfo) || !openedInfo.Mode().IsRegular() {
		_ = file.Close()
		return fmt.Errorf("driver output changed while it was opened")
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync driver output: %w", err)
	}
	if err := validateHostExecutable(file, tx.staged); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close driver output: %w", err)
	}
	currentInfo, err := tx.parent.Lstat(tx.stagedName)
	if err != nil || !os.SameFile(openedInfo, currentInfo) {
		return fmt.Errorf("driver output changed after validation")
	}
	if err := validateExistingFinal(tx.parent, tx.finalName, tx.final); err != nil {
		return err
	}
	if finalInfo, err := tx.parent.Lstat(tx.finalName); err == nil && os.SameFile(currentInfo, finalInfo) {
		return fmt.Errorf("driver output aliases existing final output")
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("reinspect driver output %q: %w", tx.final, err)
	}
	// Recheck the user-visible parent after driver execution.
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
	// Rename is the commit point; later checks are diagnostic only.
	_ = tx.checkParentPath()
	// Cleanup and directory syncing are best effort after publication.
	if !tx.keepDir {
		_ = tx.parent.Remove(tx.workName)
	}
	syncOutputParent(tx.parent)
	tx.closed = true
	_ = tx.parent.Close()
	return nil
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
