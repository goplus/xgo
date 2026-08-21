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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
