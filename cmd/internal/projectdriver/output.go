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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

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
