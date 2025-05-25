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
	"os"
	"path/filepath"
	"testing"
)

func TestPanic(t *testing.T) {
	t.Run("initEnvPanic", func(t *testing.T) {
		defer func() {
			if e := recover(); e == nil {
				t.Fatal("initEnvPanic: no panic?")
			}
		}()
		buildVersion = "v1.2"
		initEnv()
	})
	t.Run("XGOROOT panic", func(t *testing.T) {
		defer func() {
			if e := recover(); e == nil {
				t.Fatal("XGOROOT: no panic?")
			}
		}()
		defaultXGoRoot = ""
		os.Setenv(envXGOROOT, "")
		XGOROOT()
	})
}

func TestEnv(t *testing.T) {
	xgoEnv = func() (string, error) {
		wd, _ := os.Getwd()
		root := filepath.Dir(wd)
		return "v1.0.0-beta1\n2023-10-18_17-45-50\n" + root + "\n", nil
	}
	buildVersion = ""
	initEnv()
	if !Installed() {
		t.Fatal("not Installed")
	}
	if Version() != "v1.0.0-beta1" {
		t.Fatal("TestVersion failed:", Version())
	}
	buildVersion = ""
	if Version() != "v"+MainVersion+".x" {
		t.Fatal("TestVersion failed:", Version())
	}
	if BuildDate() != "2023-10-18_17-45-50" {
		t.Fatal("BuildInfo failed:", BuildDate())
	}
}
