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
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func runCapturedCommand(cmd *exec.Cmd, display string) (stdout, stderr []byte, err error) {
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		return out.Bytes(), errOut.Bytes(), commandError(display, err, errOut.String())
	}
	return out.Bytes(), errOut.Bytes(), nil
}

func commandError(name string, err error, stderr string) error {
	if message := strings.TrimSpace(stderr); message != "" {
		return fmt.Errorf("%s: %w: %s", name, err, message)
	}
	return fmt.Errorf("%s: %w", name, err)
}
