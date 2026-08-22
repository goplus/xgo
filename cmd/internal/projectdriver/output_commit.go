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
	"io/fs"
	"os"
	"runtime"
)

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
	// Cleanup and directory syncing are best effort after publication.
	if !tx.keepDir {
		_ = tx.parent.RemoveAll(tx.workName)
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
