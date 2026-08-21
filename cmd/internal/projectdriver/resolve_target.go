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
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/goplus/xgo/x/xgoprojs"
)

type targetResolution struct {
	kind             TargetKind
	original         string
	projectDir       string
	targetImportPath string
	expectedFile     string
	multiFile        bool
	unsupportedForm  string
	recursive        bool
	graph            *effectiveGraph
	graphWorkDir     string
}

func (r *Resolver) classifyTarget(ctx context.Context, target xgoprojs.Proj) (targetResolution, error) {
	switch target := target.(type) {
	case *xgoprojs.DirProj:
		return r.classifyDirectoryTarget(ctx, target)
	case *xgoprojs.FilesProj:
		return classifyFilesTarget(target)
	case *xgoprojs.PkgPathProj:
		return r.classifyPackageTarget(ctx, target)
	default:
		return targetResolution{}, ErrNotHandled
	}
}

func (r *Resolver) classifyDirectoryTarget(ctx context.Context, target *xgoprojs.DirProj) (targetResolution, error) {
	result := targetResolution{kind: TargetDirectory, original: target.Dir, graphWorkDir: r.cwd}
	candidate := target.Dir
	if hasRecursivePattern(candidate) {
		result.unsupportedForm = "directory pattern containing ..."
		result.recursive = true
		candidate = trimRecursivePattern(candidate)
	}
	projectDir, err := canonicalExistingDir(candidate)
	if err == nil {
		result.projectDir = projectDir
		result.graphWorkDir = projectDir
		return result, nil
	}
	if isMissingPathError(err) && r.policy.graph.Overlay != "" {
		result.kind, result.projectDir, result.expectedFile, result.graph, err = r.resolveOverlayLocalTarget(ctx, candidate, result.recursive)
		if err != nil {
			return targetResolution{}, err
		}
		return result, nil
	}
	if isMissingPathError(err) {
		return targetResolution{}, ErrNotHandled
	}
	return targetResolution{}, err
}

func classifyFilesTarget(target *xgoprojs.FilesProj) (targetResolution, error) {
	if len(target.Files) == 0 {
		return targetResolution{}, ErrNotHandled
	}
	file, err := canonicalExistingFile(target.Files[0])
	if err != nil {
		return targetResolution{}, ErrNotHandled
	}
	return targetResolution{
		kind:         TargetFile,
		original:     target.Files[0],
		projectDir:   filepath.Dir(file),
		expectedFile: file,
		multiFile:    len(target.Files) != 1,
		graphWorkDir: filepath.Dir(file),
	}, nil
}

func isMissingPathError(err error) bool {
	return os.IsNotExist(err) || os.IsNotExist(errors.Unwrap(err))
}
