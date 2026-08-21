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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	gomodfile "golang.org/x/mod/modfile"
)

func effectiveVendorMode(policy GraphPolicy, moduleGoMod string, parsed *gomodfile.File) (bool, error) {
	if policy.ModMode != "" {
		return policy.ModMode == modModeVendor, nil
	}
	workspace := policy.GoWork != "" && policy.GoWork != "off"
	var (
		goVersion string
		vendorDir string
	)
	if workspace {
		data, err := os.ReadFile(policy.GoWork)
		if err != nil {
			return false, err
		}
		work, err := gomodfile.ParseWork(policy.GoWork, data, nil)
		if err != nil {
			return false, err
		}
		if work.Go != nil {
			goVersion = work.Go.Version
		}
		vendorDir = filepath.Join(filepath.Dir(policy.GoWork), "vendor")
	} else {
		if parsed.Go != nil {
			goVersion = parsed.Go.Version
		}
		vendorDir = filepath.Join(filepath.Dir(moduleGoMod), "vendor")
	}
	if goVersion == "" || !versionAtLeast(goVersion, 1, 14) {
		return false, nil
	}
	info, err := os.Stat(vendorDir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, nil
	}
	vendoredWorkspace, err := vendorManifestIsForWorkspace(vendorDir)
	if err != nil {
		return false, err
	}
	return vendoredWorkspace == workspace, nil
}

// vendorManifestIsForWorkspace mirrors cmd/go's workspace marker.
// A missing modules.txt is treated as module vendor mode.
func vendorManifestIsForWorkspace(vendorDir string) (bool, error) {
	file, err := os.Open(filepath.Join(vendorDir, "modules.txt"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer file.Close()
	var buf [512]byte
	n, err := file.Read(buf[:])
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	line, _, _ := strings.Cut(string(buf[:n]), "\n")
	annotations, ok := strings.CutPrefix(line, "## ")
	if !ok {
		return false, nil
	}
	for entry := range strings.SplitSeq(annotations, ";") {
		if strings.TrimSpace(entry) == "workspace" {
			return true, nil
		}
	}
	return false, nil
}

func versionAtLeast(version string, major, minor int) bool {
	var gotMajor, gotMinor int
	if _, err := fmt.Sscanf(version, "%d.%d", &gotMajor, &gotMinor); err != nil {
		return false
	}
	return gotMajor > major || gotMajor == major && gotMinor >= minor
}

func vendorUnsupportedError(mode string) error {
	if mode == "" {
		mode = "automatic vendor mode"
	} else {
		mode = "-mod=" + mode
	}
	return fmt.Errorf("%w (%s); select -mod=readonly or -mod=mod explicitly", ErrDriverVendorUnsupported, mode)
}
