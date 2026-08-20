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
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/goplus/xgo/x/xgoprojs"
)

// DispatchResult reports whether a runtime target was selected and, if so,
// the provider process status and build output path.
type DispatchResult struct {
	Handled bool
	Status  ProcessStatus
	Output  string
}

// TryRun resolves and, when matched, runs one runtime target. It never exits
// the process; command entry points own status-to-exit translation.
func TryRun(ctx context.Context, cwd string, target xgoprojs.Proj, flags, appArgs []string, streams Streams) (DispatchResult, error) {
	if len(appArgs) != 0 && appArgs[0] == "--" {
		appArgs = appArgs[1:]
	}
	return dispatchOne(ctx, cwd, target, flags, "run", func(ctx context.Context, resolver *Resolver, rt *Runtime) (ProcessStatus, string, error) {
		status, err := resolver.Run(ctx, rt, appArgs, streams)
		return status, "", err
	})
}

// TryBuild resolves and, when matched, builds one runtime target. It never
// exits the process; command entry points own status-to-exit translation.
func TryBuild(ctx context.Context, cwd string, target xgoprojs.Proj, flags []string, output string, streams Streams) (DispatchResult, error) {
	return dispatchOne(ctx, cwd, target, flags, "build", func(ctx context.Context, resolver *Resolver, rt *Runtime) (ProcessStatus, string, error) {
		return resolver.Build(ctx, rt, output, streams)
	})
}

// TryInstall resolves all targets before creating output or starting a provider.
func TryInstall(ctx context.Context, cwd string, targets []xgoprojs.Proj, flags []string, streams Streams) (DispatchResult, error) {
	if len(targets) == 0 {
		return DispatchResult{}, fmt.Errorf("runtime provider install requires at least one target")
	}
	for _, target := range targets {
		if target == nil {
			return DispatchResult{}, fmt.Errorf("runtime provider install requires non-nil targets")
		}
	}
	resolver, ctx, err := newDispatchResolver(ctx, cwd, flags)
	if err != nil {
		return DispatchResult{}, err
	}
	runtimes := make([]*Runtime, 0, len(targets))
	for _, target := range targets {
		rt, handled, err := resolveTarget(ctx, resolver, target)
		if err != nil {
			return DispatchResult{}, err
		}
		if handled {
			runtimes = append(runtimes, rt)
		}
	}
	if len(runtimes) == 0 {
		return DispatchResult{}, nil
	}
	if len(targets) != 1 {
		return DispatchResult{}, fmt.Errorf("runtime provider v1 install accepts exactly one target")
	}
	boundary := beginRuntimeSignalBoundary(ctx)
	status, final, err := resolver.Install(boundary.Context(), runtimes[0], streams)
	return finishDispatch(boundary, DispatchResult{Output: final}, status, err)
}

func dispatchOne(ctx context.Context, cwd string, target xgoprojs.Proj, flags []string, action string, run func(context.Context, *Resolver, *Runtime) (ProcessStatus, string, error)) (DispatchResult, error) {
	if target == nil {
		return DispatchResult{}, fmt.Errorf("runtime provider %s requires a non-nil target", action)
	}
	resolver, ctx, err := newDispatchResolver(ctx, cwd, flags)
	if err != nil {
		return DispatchResult{}, err
	}
	rt, handled, err := resolveTarget(ctx, resolver, target)
	if err != nil || !handled {
		return DispatchResult{Handled: handled}, err
	}
	boundary := beginRuntimeSignalBoundary(ctx)
	status, output, err := run(boundary.Context(), resolver, rt)
	return finishDispatch(boundary, DispatchResult{Output: output}, status, err)
}

func finishDispatch(boundary *runtimeSignalBoundary, result DispatchResult, status ProcessStatus, err error) (DispatchResult, error) {
	status, err = boundary.Finish(status, err)
	if err != nil {
		return DispatchResult{}, err
	}
	result.Handled = true
	result.Status = status
	return result, nil
}

func newDispatchResolver(ctx context.Context, cwd string, flags []string) (*Resolver, context.Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var err error
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			return nil, ctx, err
		}
	}
	resolver, err := NewResolver(ctx, cwd, flags)
	if err != nil {
		return nil, ctx, err
	}
	return resolver, ctx, nil
}

func resolveTarget(ctx context.Context, resolver *Resolver, target xgoprojs.Proj) (*Runtime, bool, error) {
	if target == nil {
		return nil, false, fmt.Errorf("runtime provider dispatch requires a non-nil target")
	}
	rt, err := resolver.Resolve(ctx, target)
	if errors.Is(err, ErrNotHandled) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return rt, true, nil
}
