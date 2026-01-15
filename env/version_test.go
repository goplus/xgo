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
	"strings"
	"testing"
)

func TestPanic(t *testing.T) {
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
	// When no buildVersion is set, Version() should return the module version
	// from debug.ReadBuildInfo() or "(devel)"
	ver := Version()
	if ver != "(devel)" && !strings.HasPrefix(ver, "v") {
		t.Fatal("TestVersion failed - expected (devel) or version starting with v, got:", ver)
	}
	if BuildDate() != "2023-10-18_17-45-50" {
		t.Fatal("BuildInfo failed:", BuildDate())
	}
}

func TestMainVersion(t *testing.T) {
	tests := []struct {
		version  string
		expected string
	}{
		{"v1.5.3", "1.5"},
		{"v1.5.0-rc1", "1.5"},
		{"v2.0.10", "2.0"},
		{"(devel)", "0.0"},
		{"v0.0.0-20240101-abcdef", "0.0"},
		{"invalid", "0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			// Temporarily set buildVersion to control Version() output
			oldBuildVersion := buildVersion
			buildVersion = tt.version
			defer func() { buildVersion = oldBuildVersion }()

			result := MainVersion()
			if result != tt.expected {
				t.Fatalf("MainVersion() for %s: expected %s, got %s", tt.version, tt.expected, result)
			}
		})
	}
}
