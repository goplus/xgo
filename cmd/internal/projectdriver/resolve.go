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

	"github.com/goplus/xgo/env"
	"github.com/goplus/xgo/x/xgoprojs"
)

// Resolver owns one invocation's graph and host-build policies.
type Resolver struct {
	cwd        string
	policy     parsedFlags
	xgoVersion string
}

var currentXGoVersion = env.Version

// NewResolver snapshots ambient GOFLAGS/GOWORK once. Deferred flags are used
// only to classify legacy-compatible targets, never to execute a driver.
func NewResolver(ctx context.Context, cwd string, flags []string) (*Resolver, error) {
	cwd, err := canonicalExistingDir(cwd)
	if err != nil {
		return nil, err
	}
	policy, err := preparePolicies(ctx, cwd, flags)
	if err != nil {
		return nil, err
	}
	return &Resolver{cwd: cwd, policy: policy, xgoVersion: currentXGoVersion()}, nil
}

// Resolve resolves a parsed XGo target without parsing or generating source.
func (r *Resolver) Resolve(ctx context.Context, target xgoprojs.Proj) (*Driver, error) {
	if files, ok := target.(*xgoprojs.FilesProj); ok && len(files.Files) > 1 {
		return r.resolveFileTargets(ctx, files)
	}
	input, err := r.classifyTarget(ctx, target)
	if err != nil {
		return nil, err
	}
	return r.resolveClassifiedTarget(ctx, input)
}

func (r *Resolver) resolveFileTargets(ctx context.Context, target *xgoprojs.FilesProj) (*Driver, error) {
	for _, name := range target.Files {
		input, err := r.classifyFileTarget(name)
		if err != nil {
			return nil, err
		}
		input.multiFile = true
		driver, err := r.resolveClassifiedTarget(ctx, input)
		if !errors.Is(err, ErrNotHandled) {
			return driver, err
		}
	}
	return nil, ErrNotHandled
}

func (r *Resolver) resolveClassifiedTarget(ctx context.Context, input targetResolution) (*Driver, error) {
	policy := r.policy.graph
	policy.WorkDir = input.graphWorkDir
	graph, err := r.resolveTargetGraph(ctx, input.projectDir, input.graph, policy, input.recursive)
	if err != nil {
		return nil, err
	}
	driver, err := r.buildResolvedDriver(input, graph, policy)
	if err != nil {
		return nil, err
	}
	if err := r.policy.validateDriver(); err != nil {
		return nil, err
	}
	driver.buildPolicy = r.policy.build
	return driver, nil
}

func (r *Resolver) resolveTargetGraph(ctx context.Context, projectDir string, graph *effectiveGraph, policy GraphPolicy, recursive bool) (*effectiveGraph, error) {
	if graph != nil {
		if graph.files == nil || !graph.files.hasOverlay() {
			return graph, nil
		}
		hasDriver, err := overlayDriverProjectMatch(projectDir, graph, recursive)
		if err != nil {
			return nil, err
		}
		if !hasDriver {
			return nil, ErrNotHandled
		}
		return nil, unsupportedOverlayError(policy.Overlay)
	}
	// An overlay may change both the module graph and class metadata.
	if overlay := policy.Overlay; overlay != "" {
		overlayGraph, err := loadEffectiveGraph(ctx, projectDir, policy)
		if err != nil {
			if errors.Is(err, errNoGoModule) {
				return nil, ErrNotHandled
			}
			return nil, err
		}
		hasDriver, err := overlayDriverProjectMatch(projectDir, overlayGraph, recursive)
		if err != nil {
			return nil, err
		}
		if !hasDriver {
			return nil, ErrNotHandled
		}
		return nil, unsupportedOverlayError(overlay)
	}
	preflightModule, _, hasClass, vendor, err := r.preflightClassMetadata(ctx, projectDir)
	if err != nil {
		if errors.Is(err, errNoGoModule) {
			return nil, ErrNotHandled
		}
		return nil, err
	}
	if !hasClass {
		return nil, ErrNotHandled
	}
	if vendor {
		hasDriver, err := r.matchVendorModule(projectDir, preflightModule, recursive)
		if err != nil {
			return nil, err
		}
		if !hasDriver {
			return nil, ErrNotHandled
		}
		return nil, vendorUnsupportedError(string(r.policy.graph.ModMode))
	}
	return loadEffectiveGraph(ctx, projectDir, r.policy.graph)
}
