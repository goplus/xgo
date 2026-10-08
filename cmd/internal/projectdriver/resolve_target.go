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
	"fmt"
	"os"
	"path/filepath"

	"github.com/goplus/xgo/x/xgoprojs"
)

type targetResolution struct {
	original         string
	projectDir       string
	targetImportPath string
	expectedFile     string
	expectedFileInfo os.FileInfo
	fileError        error
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
		if len(target.Files) == 0 {
			return targetResolution{}, ErrNotHandled
		}
		return r.classifyFileTarget(target.Files[0])
	case *xgoprojs.PkgPathProj:
		return r.classifyPackageTarget(ctx, target)
	default:
		return targetResolution{}, ErrNotHandled
	}
}

func (r *Resolver) classifyDirectoryTarget(ctx context.Context, target *xgoprojs.DirProj) (targetResolution, error) {
	result := targetResolution{graphWorkDir: r.cwd}
	candidate := target.Dir
	if hasRecursivePattern(candidate) {
		result.unsupportedForm = "directory pattern containing ..."
		result.recursive = true
		candidate = trimRecursivePattern(candidate)
	}
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(r.cwd, candidate)
	}
	projectDir, err := canonicalExistingDir(candidate)
	if err == nil {
		result.projectDir = projectDir
		result.graphWorkDir = projectDir
		return result, nil
	}
	if isMissingPathError(err) && r.policy.graph.Overlay != "" {
		result.projectDir, result.graph, err = r.resolveOverlayLocalTarget(ctx, candidate, result.recursive)
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

func (r *Resolver) classifyFileTarget(name string) (targetResolution, error) {
	absolute := name
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(r.cwd, absolute)
	}
	projectDir, err := canonicalExistingDir(filepath.Dir(filepath.Clean(absolute)))
	if err != nil {
		return targetResolution{}, err
	}
	file := filepath.Join(projectDir, filepath.Base(absolute))
	fileInfo, inspectErr := os.Lstat(file)
	var fileErr error
	if inspectErr != nil {
		fileErr = fmt.Errorf("project file %q cannot be inspected: %w", name, inspectErr)
		fileInfo = nil
	} else if fileInfo.Mode()&os.ModeSymlink != 0 || !fileInfo.Mode().IsRegular() {
		fileErr = fmt.Errorf("project file %q is not a regular non-symlink file", name)
		fileInfo = nil
	}
	return targetResolution{
		original:         name,
		projectDir:       projectDir,
		expectedFile:     file,
		expectedFileInfo: fileInfo,
		fileError:        fileErr,
		graphWorkDir:     projectDir,
	}, nil
}

func isMissingPathError(err error) bool {
	return os.IsNotExist(err) || os.IsNotExist(errors.Unwrap(err))
}
