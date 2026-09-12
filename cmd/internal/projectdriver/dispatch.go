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

	"github.com/goplus/xgo/x/xgoprojs"
)

// DispatchResult reports whether a driver-backed target was selected and, if so,
// the driver process status and build output path.
type DispatchResult struct {
	Handled bool
	Status  ProcessStatus
	Output  string
}

// TryRun resolves and, when matched, runs one driver-backed target. It never exits
// the process; command entry points own status-to-exit translation.
func TryRun(ctx context.Context, cwd string, target xgoprojs.Proj, flags, appArgs []string, streams Streams) (DispatchResult, error) {
	// Consume only XGo's application separator; protocol encoding adds its own
	// wire delimiter and preserves every subsequent argument.
	if len(appArgs) != 0 && appArgs[0] == "--" {
		appArgs = appArgs[1:]
	}
	return dispatchOne(ctx, cwd, target, flags, actionRun, func(ctx context.Context, resolver *Resolver, drv *Driver) (ProcessStatus, string, error) {
		status, err := resolver.Run(ctx, drv, appArgs, streams)
		return status, "", err
	})
}

// TryBuild resolves and, when matched, builds one driver-backed target. It never
// exits the process; command entry points own status-to-exit translation.
func TryBuild(ctx context.Context, cwd string, target xgoprojs.Proj, flags []string, output string, streams Streams) (DispatchResult, error) {
	return dispatchOne(ctx, cwd, target, flags, actionBuild, func(ctx context.Context, resolver *Resolver, drv *Driver) (ProcessStatus, string, error) {
		return resolver.Build(ctx, drv, output, streams)
	})
}

// TryInstall resolves all targets before creating output or starting a driver.
func TryInstall(ctx context.Context, cwd string, targets []xgoprojs.Proj, flags []string, streams Streams) (DispatchResult, error) {
	if len(targets) == 0 {
		return DispatchResult{}, fmt.Errorf("driver install requires at least one target")
	}
	for _, target := range targets {
		if target == nil {
			return DispatchResult{}, fmt.Errorf("driver install requires non-nil targets")
		}
	}
	resolver, ctx, err := newDispatchResolver(ctx, cwd, flags)
	if err != nil {
		return DispatchResult{}, err
	}
	drivers := make([]*Driver, 0, len(targets))
	for _, target := range targets {
		drv, handled, err := resolveTarget(ctx, resolver, target)
		if err != nil {
			return DispatchResult{}, err
		}
		if handled {
			drivers = append(drivers, drv)
		}
	}
	if len(drivers) == 0 {
		return DispatchResult{}, nil
	}
	if len(targets) != 1 {
		return DispatchResult{}, fmt.Errorf("driver v1 install accepts exactly one target")
	}
	boundary := beginDriverSignalBoundary(ctx)
	status, final, err := resolver.Install(boundary.Context(), drivers[0], streams)
	return finishDispatch(boundary, status, final, err)
}

func dispatchOne(ctx context.Context, cwd string, target xgoprojs.Proj, flags []string, act action, run func(context.Context, *Resolver, *Driver) (ProcessStatus, string, error)) (DispatchResult, error) {
	if target == nil {
		return DispatchResult{}, fmt.Errorf("driver %s requires a non-nil target", act)
	}
	resolver, ctx, err := newDispatchResolver(ctx, cwd, flags)
	if err != nil {
		return DispatchResult{}, err
	}
	drv, handled, err := resolveTarget(ctx, resolver, target)
	if err != nil || !handled {
		return DispatchResult{Handled: handled}, err
	}
	boundary := beginDriverSignalBoundary(ctx)
	status, output, err := run(boundary.Context(), resolver, drv)
	return finishDispatch(boundary, status, output, err)
}

func finishDispatch(boundary *driverSignalBoundary, status ProcessStatus, output string, err error) (DispatchResult, error) {
	status, err = boundary.Finish(status, err)
	if err != nil {
		return DispatchResult{}, err
	}
	return DispatchResult{Handled: true, Status: status, Output: output}, nil
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

func resolveTarget(ctx context.Context, resolver *Resolver, target xgoprojs.Proj) (*Driver, bool, error) {
	drv, err := resolver.Resolve(ctx, target)
	if errors.Is(err, ErrNotHandled) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return drv, true, nil
}
