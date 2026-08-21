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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOutputTransactionCommit(t *testing.T) {
	dir := t.TempDir()
	final := filepath.Join(dir, "game")
	if runtime.GOOS == "windows" {
		final += ".exe"
	}
	if err := os.WriteFile(final, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	tx, err := beginOutputTransaction(final, false)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.abort()
	writeTestExecutable(t, tx.staged)
	if err := tx.commitContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(final)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("final is empty")
	}
	if _, err := os.Stat(tx.dir); !os.IsNotExist(err) {
		t.Fatalf("staging remains: %v", err)
	}
}

func writeTestExecutable(t *testing.T, target string) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, data, 0755); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "darwin" {
		if output, err := exec.Command("/usr/bin/codesign", "--force", "--sign", "-", target).CombinedOutput(); err != nil {
			t.Fatalf("sign test executable: %v: %s", err, output)
		}
	}
}

func TestOutputTransactionFailurePreservesFinal(t *testing.T) {
	dir := t.TempDir()
	final := filepath.Join(dir, "game")
	if err := os.WriteFile(final, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	tx, err := beginOutputTransaction(final, false)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.abort()
	if err := os.WriteFile(tx.staged, nil, 0755); err != nil {
		t.Fatal(err)
	}
	if err := tx.commitContext(context.Background()); err == nil {
		t.Fatal("empty staged output committed")
	}
	tx.abort()
	got, err := os.ReadFile(final)
	if err != nil || string(got) != "old" {
		t.Fatalf("final changed: %q, %v", got, err)
	}
	if _, err := os.Stat(tx.dir); !os.IsNotExist(err) {
		t.Fatalf("failed transaction left staging directory: %v", err)
	}
}

func TestCanceledBuildDoesNotCommitOutput(t *testing.T) {
	dir := t.TempDir()
	final := filepath.Join(dir, executableName("game"))
	if err := os.WriteFile(final, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	tx, err := beginOutputTransaction(final, false)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.abort()
	writeTestExecutable(t, tx.staged)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := tx.commitContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("commitContext() = %v, want context cancellation", err)
	}
	got, err := os.ReadFile(final)
	if err != nil || string(got) != "old" {
		t.Fatalf("canceled commit changed final output: %q, %v", got, err)
	}
}

func TestOutputTransactionRejectsSymlinksAndExtras(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := beginOutputTransaction(link, false); err == nil {
		t.Fatal("symlink final accepted")
	}

	final := filepath.Join(dir, "game")
	tx, err := beginOutputTransaction(final, false)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.abort()
	if err := os.WriteFile(tx.staged, []byte("new"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tx.dir, "extra"), []byte("extra"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tx.commitContext(context.Background()); err == nil {
		t.Fatal("extra staged output accepted")
	}
	if _, err := os.Stat(final); !os.IsNotExist(err) {
		t.Fatalf("final unexpectedly exists: %v", err)
	}
}

func TestOutputTransactionRejectsFinalSymlinkCreatedAfterBegin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	dir := t.TempDir()
	final := filepath.Join(dir, "game")
	tx, err := beginOutputTransaction(final, false)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.abort()
	writeTestExecutable(t, tx.staged)
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("target"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, final); err != nil {
		t.Fatal(err)
	}
	if err := tx.commitContext(context.Background()); err == nil {
		t.Fatal("final symlink created after begin was accepted")
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "target" {
		t.Fatalf("symlink target changed: %q, %v", got, err)
	}
}

func TestOutputTransactionPinsParentAcrossPathSwap(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "out")
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	final := filepath.Join(parent, "game")
	if runtime.GOOS == "windows" {
		final += ".exe"
	}
	if err := os.WriteFile(final, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	tx, err := beginOutputTransaction(final, false)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.abort()
	writeTestExecutable(t, tx.staged)

	moved := filepath.Join(base, "moved")
	if err := os.Rename(parent, moved); err != nil {
		t.Skipf("platform/filesystem cannot rename an open pinned directory: %v", err)
	}
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	replacementFinal := filepath.Join(parent, filepath.Base(final))
	if err := os.WriteFile(replacementFinal, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}

	commitErr := tx.commitContext(context.Background())
	if commitErr == nil {
		t.Fatal("commit accepted a replaced output parent pathname")
	}
	if got, err := os.ReadFile(replacementFinal); err != nil || string(got) != "replacement" {
		t.Fatalf("replacement parent was modified: %q, %v", got, err)
	}
	movedFinal := filepath.Join(moved, filepath.Base(final))
	got, err := os.ReadFile(movedFinal)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Fatalf("failed commit changed pinned output: %q (%v)", got, commitErr)
	}
}

func TestOutputTransactionRejectsWorkDirectorySwap(t *testing.T) {
	dir := t.TempDir()
	final := filepath.Join(dir, "game")
	if runtime.GOOS == "windows" {
		final += ".exe"
	}
	tx, err := beginOutputTransaction(final, false)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.abort()
	moved := tx.dir + ".moved"
	if err := os.Rename(tx.dir, moved); err != nil {
		t.Skipf("platform/filesystem cannot rename an open work directory: %v", err)
	}
	if err := os.Mkdir(tx.dir, 0700); err != nil {
		t.Fatal(err)
	}
	writeTestExecutable(t, tx.staged)
	if err := tx.commitContext(context.Background()); err == nil {
		t.Fatal("replaced work directory was accepted")
	}
	if _, err := os.Stat(final); !os.IsNotExist(err) {
		t.Fatalf("failed work-directory transaction published output: %v", err)
	}
}

func TestOutputTransactionSupportsStableParentAlias(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	base := t.TempDir()
	parent := filepath.Join(base, "real")
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(base, "alias")
	if err := os.Symlink(parent, alias); err != nil {
		t.Fatal(err)
	}
	tx, err := beginOutputTransaction(filepath.Join(alias, "game"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.abort()
	writeTestExecutable(t, tx.staged)
	if err := tx.commitContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(parent, "game")); err != nil || info.Size() == 0 {
		t.Fatalf("aliased output = %#v, %v", info, err)
	}
}

func TestOutputTransactionWindowsCaseAlias(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows case-insensitive output semantics")
	}
	dir := t.TempDir()
	existing := filepath.Join(dir, "game.exe")
	if err := os.WriteFile(existing, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	tx, err := beginOutputTransaction(filepath.Join(dir, "GAME.EXE"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.abort()
	writeTestExecutable(t, tx.staged)
	if err := tx.commitContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(existing)
	if err != nil || info.Size() <= int64(len("old")) {
		t.Fatalf("case-aliased output = %#v, %v", info, err)
	}
}

func TestResolveBuildOutput(t *testing.T) {
	dir := t.TempDir()
	got, err := resolveBuildOutput(dir, "", "game")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, executableName("game"))
	if got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	got, err = resolveBuildOutput(dir, dir+string(filepath.Separator), "game")
	if err != nil || got != want {
		t.Fatalf("directory output = %q, %v", got, err)
	}

	got, err = resolveBuildOutput(dir, "relative/bin/game", "default")
	want = filepath.Join(dir, "relative", "bin", "game")
	if err != nil || got != want {
		t.Fatalf("relative output = %q, %v; want %q", got, err, want)
	}

	relativeDir := filepath.Join(dir, "relative-dir")
	if err := os.Mkdir(relativeDir, 0755); err != nil {
		t.Fatal(err)
	}
	got, err = resolveBuildOutput(dir, filepath.Join("relative-dir", ""), "default")
	want = filepath.Join(relativeDir, executableName("default"))
	if err != nil || got != want {
		t.Fatalf("relative directory output = %q, %v; want %q", got, err, want)
	}
}
