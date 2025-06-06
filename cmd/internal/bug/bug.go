/*
 * Copyright (c) 2021-2021 The XGo Authors (xgo.dev). All rights reserved.
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

package bug

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	urlpkg "net/url"

	"github.com/goplus/xgo/cmd/internal/base"
	"github.com/goplus/xgo/env"
)

// -----------------------------------------------------------------------------

// Cmd - xgo bug
var Cmd = &base.Command{
	UsageLine: "xgo bug",
	Short:     "Start a bug report",
}

func init() {
	Cmd.Run = runCmd
}

func runCmd(_ *base.Command, args []string) {
	if len(args) > 0 {
		log.Fatalf("xgo: bug takes no arguments")
	}
	var buf bytes.Buffer
	buf.WriteString(bugHeader)
	printXgoVersion(&buf)
	buf.WriteString("### Does this issue reproduce with the latest release?\n\n\n")
	printEnvDetails(&buf)
	buf.WriteString(bugFooter)

	body := buf.String()
	url := "https://github.com/goplus/xgo/issues/new?body=" + urlpkg.QueryEscape(body)
	if !open(url) {
		fmt.Print("Please file a new issue at golang.org/issue/new using this template:\n\n")
		fmt.Print(body)
	}
}

// -----------------------------------------------------------------------------

const bugHeader = `<!-- Please answer these questions before submitting your issue. Thanks! -->

`
const bugFooter = `### What did you do?

<!--
If possible, provide a recipe for reproducing the error.
A complete runnable program is good.
A link on play.xgo.dev is best.
-->



### What did you expect to see?



### What did you see instead?

`

func printXgoVersion(w io.Writer) {
	fmt.Fprintf(w, "### What version of xgo are you using (`xgo version`)?\n\n")
	fmt.Fprintf(w, "<pre>\n")
	fmt.Fprintf(w, "$ xgo version\n")
	fmt.Fprintf(w, "xgo version %s %s/%s\n", env.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(w, "</pre>\n")
	fmt.Fprintf(w, "\n")
}

func printEnvDetails(w io.Writer) {
	fmt.Fprintf(w, "### What operating system and processor architecture are you using (`xgo env`)?\n\n")
	fmt.Fprintf(w, "<details><summary><code>xgo env</code> Output</summary><br><pre>\n")
	fmt.Fprintf(w, "$ xgo env\n")
	printXgoEnv(w)
	printXgoDetails(w)
	printOSDetails(w)
	printCDetails(w)
	fmt.Fprintf(w, "</pre></details>\n\n")
}

func printXgoEnv(w io.Writer) {
	cmd := exec.Command("xgo", "env")
	cmd.Env = os.Environ()
	cmd.Stdout = w

	err := cmd.Run()
	if err != nil {
		log.Fatalln("run xgo env failed:", err)
	}
}

func printXgoDetails(w io.Writer) {
	xgocmd := filepath.Join(runtime.GOROOT(), "bin/xgo")
	printCmdOut(w, "GOROOT/bin/xgo version: ", xgocmd, "version")
	printCmdOut(w, "GOROOT/bin/xgo tool compile -V: ", xgocmd, "tool", "compile", "-V")
}

func printOSDetails(w io.Writer) {
	switch runtime.GOOS {
	case "darwin", "ios":
		printCmdOut(w, "uname -v: ", "uname", "-v")
		printCmdOut(w, "", "sw_vers")
	case "linux":
		printCmdOut(w, "uname -sr: ", "uname", "-sr")
		printCmdOut(w, "", "lsb_release", "-a")
		printGlibcVersion(w)
	case "openbsd", "netbsd", "freebsd", "dragonfly":
		printCmdOut(w, "uname -v: ", "uname", "-v")
	case "illumos", "solaris":
		// Be sure to use the OS-supplied uname, in "/usr/bin":
		printCmdOut(w, "uname -srv: ", "/usr/bin/uname", "-srv")
		out, err := os.ReadFile("/etc/release")
		if err == nil {
			fmt.Fprintf(w, "/etc/release: %s\n", out)
		}
	}
}

func printCDetails(w io.Writer) {
	printCmdOut(w, "lldb --version: ", "lldb", "--version")
	cmd := exec.Command("gdb", "--version")
	out, err := cmd.Output()
	if err == nil {
		// There's apparently no combination of command line flags
		// to get gdb to spit out its version without the license and warranty.
		// Print up to the first newline.
		fmt.Fprintf(w, "gdb --version: %s\n", firstLine(out))
	}
}

// printCmdOut prints the output of running the given command.
// It ignores failures; 'xgo bug' is best effort.
func printCmdOut(w io.Writer, prefix, path string, args ...string) {
	cmd := exec.Command(path, args...)
	out, err := cmd.Output()
	if err != nil {
		return
	}
	fmt.Fprintf(w, "%s%s\n", prefix, bytes.TrimSpace(out))
}

// firstLine returns the first line of a given byte slice.
func firstLine(buf []byte) []byte {
	idx := bytes.IndexByte(buf, '\n')
	if idx > 0 {
		buf = buf[:idx]
	}
	return bytes.TrimSpace(buf)
}

// printGlibcVersion prints information about the glibc version.
// It ignores failures.
func printGlibcVersion(w io.Writer) {
	tempdir := os.TempDir()
	if tempdir == "" {
		return
	}
	src := []byte(`int main() {}`)
	srcfile := filepath.Join(tempdir, "go-bug.c")
	outfile := filepath.Join(tempdir, "go-bug")
	err := os.WriteFile(srcfile, src, 0644)
	if err != nil {
		return
	}
	defer os.Remove(srcfile)
	cmd := exec.Command("gcc", "-o", outfile, srcfile)
	if _, err = cmd.CombinedOutput(); err != nil {
		return
	}
	defer os.Remove(outfile)

	cmd = exec.Command("ldd", outfile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	re := regexp.MustCompile(`libc\.so[^ ]* => ([^ ]+)`)
	m := re.FindStringSubmatch(string(out))
	if m == nil {
		return
	}
	cmd = exec.Command(m[1])
	out, err = cmd.Output()
	if err != nil {
		return
	}
	fmt.Fprintf(w, "%s: %s\n", m[1], firstLine(out))

	// print another line (the one containing version string) in case of musl libc
	if idx := bytes.IndexByte(out, '\n'); bytes.Contains(out, []byte("musl")) && idx > -1 {
		fmt.Fprintf(w, "%s\n", firstLine(out[idx+1:]))
	}
}

// Commands returns a list of possible commands to use to open a url.
func commands() [][]string {
	var cmds [][]string
	if exe := os.Getenv("BROWSER"); exe != "" {
		cmds = append(cmds, []string{exe})
	}
	switch runtime.GOOS {
	case "darwin":
		cmds = append(cmds, []string{"/usr/bin/open"})
	case "windows":
		cmds = append(cmds, []string{"cmd", "/c", "start"})
	default:
		if os.Getenv("DISPLAY") != "" {
			// xdg-open is only for use in a desktop environment.
			cmds = append(cmds, []string{"xdg-open"})
		}
	}
	cmds = append(cmds,
		[]string{"chrome"},
		[]string{"google-chrome"},
		[]string{"chromium"},
		[]string{"firefox"},
	)
	return cmds
}

// Open tries to open url in a browser and reports whether it succeeded.
func open(url string) bool {
	for _, args := range commands() {
		cmd := exec.Command(args[0], append(args[1:], url)...)
		if cmd.Start() == nil && appearsSuccessful(cmd, 3*time.Second) {
			return true
		}
	}
	return false
}

// appearsSuccessful reports whether the command appears to have run successfully.
// If the command runs longer than the timeout, it's deemed successful.
// If the command runs within the timeout, it's deemed successful if it exited cleanly.
func appearsSuccessful(cmd *exec.Cmd, timeout time.Duration) bool {
	errc := make(chan error, 1)
	go func() {
		errc <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		return true
	case err := <-errc:
		return err == nil
	}
}
