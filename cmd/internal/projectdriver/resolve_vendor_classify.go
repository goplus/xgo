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

	"github.com/goplus/mod/modfile"
	"github.com/goplus/mod/modload"
)

func (r *Resolver) vendorClassMetadataError(modulePath string) error {
	return fmt.Errorf("%w: class module %q metadata is not represented by standard Go vendor data", vendorUnsupportedError(string(r.policy.graph.ModMode)), modulePath)
}

func (r *Resolver) matchVendorModule(projectDir string, module modload.Module, recursive bool) (bool, error) {
	if classPath := externalClassModule(module); classPath != "" {
		return false, r.vendorClassMetadataError(classPath)
	}
	return matchVendorProjects(projectDir, module.Projects(), recursive)
}

func matchVendorProjects(dir string, projects []*modfile.Project, recursive bool) (bool, error) {
	if recursive {
		return patternContainsDriverProjects(dir, projects)
	}
	return hasDriverProject(dir, projects)
}

func hasDriverProject(dir string, projects []*modfile.Project) (bool, error) {
	if len(projects) == 0 {
		return false, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return false, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if driverProjectMatches(projects, entry.Name()) {
			return true, nil
		}
	}
	return false, nil
}
