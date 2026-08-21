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

	"github.com/goplus/xgo/env"
	"github.com/goplus/xgo/x/xgoprojs"
)

// Resolver owns one invocation's graph and host-build policies.
type Resolver struct {
	cwd        string
	policy     parsedFlags
	xgoVersion string
}

// NewResolver snapshots ambient GOFLAGS/GOWORK once for an invocation. Policy
// setup must not fall back to a different module graph: doing so could classify
// a workspace driver-backed target as legacy and incorrectly dispatch it to GenGo.
func NewResolver(ctx context.Context, cwd string, flags []string) (*Resolver, error) {
	cwd, err := canonicalExistingDir(cwd)
	if err != nil {
		return nil, err
	}
	policy, err := preparePolicies(ctx, cwd, flags)
	if err != nil {
		return nil, err
	}
	return &Resolver{cwd: cwd, policy: policy, xgoVersion: env.Version()}, nil
}

// BuildPolicy returns the validated driver build policy. Callers must invoke
// it only after Resolve matched a driver-backed project.
func (r *Resolver) BuildPolicy() (BuildPolicy, error) {
	if err := r.policy.validateDriver(); err != nil {
		return BuildPolicy{}, err
	}
	return r.policy.build, nil
}

// Resolve resolves a parsed XGo target without parsing or generating source.
func (r *Resolver) Resolve(ctx context.Context, target xgoprojs.Proj) (*Driver, error) {
	input, err := r.classifyTarget(ctx, target)
	if err != nil {
		return nil, err
	}
	policy := r.policy.graph
	policy.WorkDir = input.graphWorkDir
	graph, err := r.resolveTargetGraph(ctx, input.projectDir, input.graph, policy, input.recursive)
	if err != nil {
		return nil, err
	}
	return r.buildResolvedDriver(input, graph, policy)
}
