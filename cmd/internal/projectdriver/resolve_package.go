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
	"strings"

	"github.com/goplus/mod/modload"
	"github.com/goplus/xgo/x/xgoprojs"
)

func (r *Resolver) classifyPackageTarget(ctx context.Context, target *xgoprojs.PkgPathProj) (targetResolution, error) {
	result := targetResolution{
		kind:             TargetPackage,
		original:         target.Path,
		targetImportPath: target.Path,
		graphWorkDir:     r.cwd,
	}
	lookupPath := target.Path
	if strings.Contains(lookupPath, "@") {
		hasDriver, err := r.versionedPackageHasDriver(ctx, lookupPath)
		if err != nil {
			return targetResolution{}, err
		}
		if !hasDriver {
			return targetResolution{}, ErrNotHandled
		}
		return targetResolution{}, fmt.Errorf("driver v1 does not support package target containing @version")
	}
	if hasRecursivePattern(lookupPath) {
		result.unsupportedForm = "package pattern containing ..."
		result.recursive = true
		lookupPath = trimRecursivePattern(lookupPath)
	}

	hasOverlay := r.policy.graph.Overlay != ""
	var (
		preflightModule modload.Module
		preflightGoMod  string
		callerHasClass  bool
		vendor          bool
	)
	if !hasOverlay {
		var preflightErr error
		preflightModule, preflightGoMod, callerHasClass, vendor, preflightErr = r.preflightClassMetadata(ctx, r.cwd)
		if preflightErr != nil && !errors.Is(preflightErr, errNoGoModule) {
			return targetResolution{}, preflightErr
		}
		if vendor {
			if !callerHasClass && r.policy.graph.GoWork == "off" {
				return targetResolution{}, ErrNotHandled
			}
			hasDriver, probeErr := r.probeVendorPackage(ctx, preflightGoMod, preflightModule, lookupPath, result.recursive)
			if probeErr != nil {
				return targetResolution{}, probeErr
			}
			if hasDriver {
				return targetResolution{}, vendorUnsupportedError(string(r.policy.graph.ModMode))
			}
			return targetResolution{}, ErrNotHandled
		}
	}

	callerGraph, err := loadEffectiveGraph(ctx, r.cwd, r.policy.graph)
	if err != nil {
		if errors.Is(err, errNoGoModule) && !callerHasClass && !hasOverlay {
			return targetResolution{}, ErrNotHandled
		}
		return targetResolution{}, err
	}
	projectDir, targetModule, err := resolvePackageDirectory(ctx, callerGraph, lookupPath, r.cwd, r.policy.graph)
	if err != nil {
		if callerHasClass || hasOverlay {
			return targetResolution{}, err
		}
		return targetResolution{}, ErrNotHandled
	}
	if !targetModule.Main && !classModuleMarked(callerGraph.ClassModules, targetModule.Selected.Path) {
		return targetResolution{}, ErrNotHandled
	}
	result.projectDir = projectDir
	result.graph, err = retargetEffectiveGraph(callerGraph, targetModule)
	if err != nil {
		return targetResolution{}, err
	}
	return result, nil
}
