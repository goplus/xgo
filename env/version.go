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
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
)

// EnvKey represents environment variable keys
type EnvKey = string

// Environment variable keys
const (
	EnvKeyBuildDate  EnvKey = "BUILDDATE"
	EnvKeyXGoRoot    EnvKey = "XGOROOT"
	EnvKeyXGoVersion EnvKey = "XGOVERSION"
	EnvKeyXGoGoCmd   EnvKey = "XGO_GOCMD"
	EnvKeyGoModCache EnvKey = "GOMODCACHE"
	EnvKeyGoXMod     EnvKey = "GOXMOD"
	EnvKeyHome       EnvKey = "HOME"
)

// buildVersion is the XGo tree's version string at build time.
// This is set by the linker via ldflags for official releases.
var (
	buildVersion string
)

func init() {
	initEnv()
}

func initEnv() {
	if buildVersion == "" {
		initEnvByXgo()
	}
}

func initEnvByXgo() {
	if fname := filepath.Base(os.Args[0]); !isXgoCmd(fname) {
		if ret, err := xgoEnv(); err == nil {
			parts := strings.SplitN(strings.TrimRight(ret, "\n"), "\n", 3)
			if len(parts) == 3 {
				buildVersion, buildDate, defaultXGoRoot = parts[0], parts[1], parts[2]
			}
		}
	}
}

var xgoEnv = func() (string, error) {
	var b bytes.Buffer
	cmd := exec.Command("xgo", "env", EnvKeyXGoVersion, EnvKeyBuildDate, EnvKeyXGoRoot)
	cmd.Stdout = &b
	err := cmd.Run()
	return b.String(), err
}

// Installed checks is `xgo` installed or not.
// If returns false, it means `xgo` is not installed or not in PATH.
func Installed() bool {
	return buildVersion != ""
}

// Version returns the XGo tree's version string.
// It is either the commit hash and date at the time of the build or,
// when possible, a release tag like "v1.0.0-rc1".
//
// Version detection priority:
// 1. buildVersion from ldflags (for official releases via goreleaser)
// 2. debug.ReadBuildInfo() - reads embedded Go module version from VCS
// 3. "(devel)" for non-VCS builds
func Version() string {
	// Prefer ldflags-injected version (for official releases)
	if buildVersion != "" {
		return buildVersion
	}

	// Fallback to debug.ReadBuildInfo (embedded module version from VCS)
	if bi, ok := debug.ReadBuildInfo(); ok {
		if bi.Main.Version != "" {
			return bi.Main.Version
		}
	}

	// Return devel for non-VCS builds
	return "(devel)"
}

// MainVersion extracts the major.minor version from the current version.
// For example, "v1.5.3" returns "1.5", "v1.5.0-rc1" returns "1.5".
// For development versions like "(devel)" or pseudo-versions, returns "0.0".
func MainVersion() string {
	ver := Version()

	// Handle (devel) and other non-version strings
	if !strings.HasPrefix(ver, "v") {
		return "0.0"
	}

	// Remove 'v' prefix
	ver = strings.TrimPrefix(ver, "v")

	// Handle pseudo-versions (e.g., "v0.0.0-20240101-abcdef")
	if strings.HasPrefix(ver, "0.0.0-") {
		return "0.0"
	}

	// Extract major.minor from semantic version
	parts := strings.Split(ver, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}

	return "0.0"
}
