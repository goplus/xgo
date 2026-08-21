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
)

func (r *Resolver) buildResolvedDriver(target targetResolution, graph *effectiveGraph, policy GraphPolicy) (*Driver, error) {
	module, hasClass, err := loadResolvedClasses(graph)
	if err != nil {
		return nil, err
	}
	if !hasClass {
		return nil, ErrNotHandled
	}
	if target.recursive {
		hasDriver, err := patternContainsDriverProject(target.projectDir, module)
		if err != nil {
			return nil, err
		}
		if !hasDriver {
			return nil, ErrNotHandled
		}
		return nil, fmt.Errorf("driver v1 does not support %s", target.unsupportedForm)
	}

	projectFile, info, candidates, err := findProjectFile(target.projectDir, module)
	if err != nil {
		return nil, err
	}
	if info == nil || info.Project.Driver == nil {
		return nil, ErrNotHandled
	}
	if target.unsupportedForm != "" {
		return nil, fmt.Errorf("driver v1 does not support %s", target.unsupportedForm)
	}
	if candidates != 1 {
		return nil, fmt.Errorf("driver-backed project directory %q contains %d project files; exactly one is required", target.projectDir, candidates)
	}
	if target.multiFile {
		return nil, fmt.Errorf("driver v1 does not support multiple source-file targets")
	}
	if target.expectedFile != "" {
		same, err := sameFile(target.expectedFile, projectFile)
		if err != nil || !same {
			return nil, fmt.Errorf("driver file target %q is not the unique project file %q", target.original, projectFile)
		}
	}
	if info.Origin == nil {
		return nil, fmt.Errorf("driver-backed project %q has no module provenance", projectFile)
	}
	if err := checkRequiredXGo(info.RequiredXGo, r.xgoVersion); err != nil {
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
	goxmod, err := declaringMetadata(origin, info.Declaration)
	if err != nil {
		return nil, err
	}
	if os.Getenv("XGO_DRIVER") == "off" {
		return nil, ErrDriverDisabled
	}
	if os.Getenv(driverGuardEnv) != "" {
		return nil, ErrDriverRecursive
	}
	defaultName := defaultExecutableName(target.kind, target.projectDir, target.targetImportPath)
	return &Driver{
		TargetKind:       target.kind,
		OriginalTarget:   target.original,
		TargetImportPath: target.targetImportPath,
		DefaultExecName:  defaultName,
		ProjectDir:       target.projectDir,
		ProjectFile:      projectFile,
		ModuleRoot:       graph.Target.Effective().Dir,
		DriverPackage:    project.Driver.Package,
		Origin:           origin,
		RequiredXGo:      info.RequiredXGo,
		Protocol:         project.Driver.Protocol,
		ProjectExt:       project.Ext,
		ProjectFullExt:   project.FullExt,
		PackDir:          packDir,
		PackIndex:        packIndex,
		GoxMod:           goxmod.Path,
		GoxModSHA256:     goxmod.SHA256,
		Graph:            policy,
	}, nil
}
