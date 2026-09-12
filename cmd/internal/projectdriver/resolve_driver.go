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
	"io"
	"os"
	"path/filepath"

	"github.com/goplus/mod/xgomod"
)

func (r *Resolver) buildResolvedDriver(target targetResolution, graph *effectiveGraph, policy GraphPolicy) (*Driver, error) {
	module, hasClass, err := loadResolvedClasses(graph)
	if err != nil {
		return nil, err
	}
	if !hasClass {
		return nil, ErrNotHandled
	}
	projectFile, info, err := resolveDriverProject(target, module)
	if err != nil {
		return nil, err
	}
	if err := checkDriverXGoVersion(info.RequiredXGo, r.xgoVersion); err != nil {
		return nil, err
	}
	project := info.Project
	if project.Driver.Protocol != protocolV1 {
		return nil, fmt.Errorf("unsupported driver protocol %q", project.Driver.Protocol)
	}
	packDir, packIndex, err := validatePack(target.projectDir, project.Pack)
	if err != nil {
		return nil, err
	}
	origin := *info.Origin
	declaration, err := declaringMetadata(origin, info.Declaration)
	if err != nil {
		return nil, err
	}
	if os.Getenv("XGO_DRIVER") == "off" {
		return nil, ErrDriverDisabled
	}
	if os.Getenv(driverGuardEnv) != "" {
		return nil, ErrDriverRecursive
	}
	return &Driver{
		DefaultExecName: defaultExecutableName(target.projectDir, target.targetImportPath),
		ProjectDir:      target.projectDir,
		ProjectFile:     projectFile,
		ModuleRoot:      graph.Target.Effective().Dir,
		DriverPackage:   project.Driver.Package,
		Origin:          origin,
		ProjectExt:      project.Ext,
		ProjectFullExt:  project.FullExt,
		PackDir:         packDir,
		PackIndex:       packIndex,
		Declaration:     declaration,
		TargetModFile:   graph.TargetModFile,
		Graph:           policy,
	}, nil
}

func resolveDriverProject(target targetResolution, module *xgomod.Module) (string, *xgomod.ProjectInfo, error) {
	if target.recursive {
		hasDriver, err := patternContainsDriverProject(target.projectDir, module)
		if err != nil {
			return "", nil, err
		}
		if !hasDriver {
			return "", nil, ErrNotHandled
		}
		return "", nil, fmt.Errorf("driver v1 does not support %s", target.unsupportedForm)
	}

	projectFile, info, candidates, err := findProjectFile(target.projectDir, module)
	if err != nil {
		return "", nil, err
	}
	if info == nil || info.Project.Driver == nil {
		return "", nil, ErrNotHandled
	}
	if candidates != 1 {
		return "", nil, fmt.Errorf("driver-backed project directory %q contains %d project files; exactly one is required", target.projectDir, candidates)
	}
	if target.multiFile {
		return "", nil, fmt.Errorf("driver v1 does not support multiple source-file targets")
	}
	if target.fileError != nil {
		return "", nil, target.fileError
	}
	if err := validateDriverFileTarget(target, projectFile); err != nil {
		return "", nil, err
	}
	if info.Origin == nil {
		return "", nil, fmt.Errorf("driver-backed project %q has no module provenance", projectFile)
	}
	return projectFile, info, nil
}

func validateDriverFileTarget(target targetResolution, projectFile string) error {
	if target.expectedFile != "" {
		if target.expectedFileInfo != nil {
			current, err := os.Lstat(target.expectedFile)
			if err != nil || current.Mode()&os.ModeSymlink != 0 || !current.Mode().IsRegular() || !os.SameFile(target.expectedFileInfo, current) {
				return fmt.Errorf("driver file target %q changed during classification", target.original)
			}
		}
		same, err := sameFile(target.expectedFile, projectFile)
		if err != nil || !same {
			return fmt.Errorf("driver file target %q is not the unique project file %q", target.original, projectFile)
		}
	}
	return nil
}

func declaringMetadata(origin ResolvedModule, snapshot xgomod.FileIdentity) (fileIdentity, error) {
	dir := origin.Effective().Dir
	base := filepath.Base(snapshot.Path)
	if snapshot.Path == "" || snapshot.SHA256 == "" || filepath.Dir(snapshot.Path) != dir || (base != "gox.mod" && base != "gop.mod") {
		return fileIdentity{}, fmt.Errorf("driver origin %q has an invalid declaring metadata snapshot", origin.Selected.Path)
	}
	if err := verifyFileSnapshot("declaring metadata", snapshot); err != nil {
		return fileIdentity{}, err
	}
	return fileIdentity(snapshot), nil
}

func verifyTargetModFile(snapshot xgomod.FileIdentity) error {
	if snapshot.Path == "" || snapshot.SHA256 == "" || !filepath.IsAbs(snapshot.Path) {
		return fmt.Errorf("invalid target modfile snapshot")
	}
	return verifyFileSnapshot("target modfile", snapshot)
}

func verifyFileSnapshot(label string, snapshot xgomod.FileIdentity) error {
	path := snapshot.Path
	before, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%s %q: %w", label, path, err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return fmt.Errorf("%s %q is not a regular non-symlink file", label, path)
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%s %q: %w", label, path, err)
	}
	data, readErr := io.ReadAll(file)
	opened, statErr := file.Stat()
	closeErr := file.Close()
	if readErr != nil {
		return readErr
	}
	if statErr != nil {
		return statErr
	}
	if closeErr != nil {
		return closeErr
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(before, opened) || !os.SameFile(opened, after) || !after.Mode().IsRegular() {
		return fmt.Errorf("%s %q changed after discovery", label, path)
	}
	if got := sha256Bytes(data); got != snapshot.SHA256 {
		return fmt.Errorf("%s %q changed after discovery", label, path)
	}
	return nil
}
