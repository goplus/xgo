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

package runtimeprovider

import (
	"runtime"
	"strings"
	"testing"
)

func TestHostBuildEnvironmentPreservesCGO(t *testing.T) {
	env := hostBuildEnvironment([]string{
		"GOOS=target-os",
		"GOARCH=target-arch",
		"CGO_ENABLED=1",
	}, "off")
	if got, _ := environmentValue(env, "GOOS"); got != runtime.GOOS {
		t.Fatalf("GOOS = %q, want host %q", got, runtime.GOOS)
	}
	if got, _ := environmentValue(env, "GOARCH"); got != runtime.GOARCH {
		t.Fatalf("GOARCH = %q, want host %q", got, runtime.GOARCH)
	}
	if got, ok := environmentValue(env, "CGO_ENABLED"); !ok || got != "1" {
		t.Fatalf("CGO_ENABLED = %q, %t; want inherited value 1", got, ok)
	}

	env = hostBuildEnvironment([]string{"PATH=/bin"}, "off")
	if got, ok := environmentValue(env, "CGO_ENABLED"); ok {
		t.Fatalf("CGO_ENABLED = %q; host default should remain unset", got)
	}
}

func environmentValue(env []string, key string) (string, bool) {
	for _, entry := range env {
		name, value, ok := strings.Cut(entry, "=")
		if ok && name == key {
			return value, true
		}
	}
	return "", false
}
