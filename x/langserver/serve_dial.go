/*
 * Copyright (c) 2023 The XGo Authors (xgo.dev). All rights reserved.
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

package langserver

import (
	"context"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/goplus/xgo/x/jsonrpc2/stdio"
)

func fatal(err error) {
	log.Fatalln(err)
}

// -----------------------------------------------------------------------------

// ServeAndDialConfig represents the configuration of ServeAndDial.
type ServeAndDialConfig struct {
	// OnError is to customize how to process errors (optional).
	// It should panic in any case.
	OnError func(err error)
}

const (
	logPrefix = "serve-"
	logSuffix = ".log"
)

func logFileOf(xgoDir string, pid int) string {
	return xgoDir + logPrefix + strconv.Itoa(pid) + logSuffix
}

func isLog(fname string) bool {
	return strings.HasSuffix(fname, logSuffix) && strings.HasPrefix(fname, logPrefix)
}

func pidByName(fname string) int {
	pidText := fname[len(logPrefix) : len(fname)-len(logSuffix)]
	if ret, err := strconv.ParseInt(pidText, 10, 0); err == nil {
		return int(ret)
	}
	return -1
}

func killByPid(pid int) {
	if pid < 0 {
		return
	}
	if proc, err := os.FindProcess(pid); err == nil {
		proc.Kill()
	}
}

func tooOld(d fs.DirEntry) bool {
	if fi, e := d.Info(); e == nil {
		lastHour := time.Now().Add(-time.Hour)
		return fi.ModTime().Before(lastHour)
	}
	return false
}

// ServeAndDial executes a command as a LangServer, makes a new connection to it
// and returns a client of the LangServer based on the connection.
func ServeAndDial(conf *ServeAndDialConfig, xgoCmd string, args ...string) Client {
	if conf == nil {
		conf = new(ServeAndDialConfig)
	}
	onErr := conf.OnError
	if onErr == nil {
		onErr = fatal
	}

	home, err := os.UserHomeDir()
	if err != nil {
		onErr(err)
	}
	xgoDir := home + "/.xgo/"
	err = os.MkdirAll(xgoDir, 0755)
	if err != nil {
		onErr(err)
	}

	// logFile is where the LangServer application log saves to.
	// default is ~/.xgo/serve-{pid}.log
	logFile := logFileOf(xgoDir, os.Getpid())

	// clean too old logfiles, and kill old LangServer processes
	go func() {
		if fis, e := os.ReadDir(xgoDir); e == nil {
			for _, fi := range fis {
				if fi.IsDir() {
					continue
				}
				if fname := fi.Name(); isLog(fname) && tooOld(fi) {
					os.Remove(xgoDir + fname)
					killByPid(pidByName(fname))
				}
			}
		}
	}()

	in, w := io.Pipe()
	r, out := io.Pipe()

	var cmd *exec.Cmd
	go func() {
		defer r.Close()
		defer w.Close()

		f, err := os.Create(logFile)
		if err != nil {
			onErr(err)
		}
		defer f.Close()

		cmd = exec.Command(xgoCmd, args...)
		cmd.Stdin = r
		cmd.Stdout = w
		cmd.Stderr = f
		err = cmd.Start()
		if err != nil {
			onErr(err)
		}

		newLogFile := logFileOf(xgoDir, cmd.Process.Pid)
		os.Rename(logFile, newLogFile)

		cmd.Wait()
	}()

	c, err := Open(context.Background(), stdio.Dialer(in, out), func() {
		if cmd == nil {
			return
		}
		if proc := cmd.Process; proc != nil {
			log.Println("==> ServeAndDial: kill", proc.Pid, xgoCmd, args)
			proc.Kill()
		}
	})
	if err != nil {
		onErr(err)
	}
	return c
}

// -----------------------------------------------------------------------------
