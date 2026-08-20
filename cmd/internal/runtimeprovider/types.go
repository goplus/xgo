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

// Package runtimeprovider implements XGo's private runtime-provider
// dispatcher. The shared argv contract lives in mod/runtimeprotocol so XGo
// and provider implementations cannot maintain divergent codecs.
package runtimeprovider

import (
	"errors"
	"io"
	"os"

	"github.com/goplus/mod/runtimeprotocol"
	"github.com/goplus/mod/xgomod"
)

const protocolV1 = runtimeprotocol.Version1

var (
	// ErrNotHandled means that the target is not backed by a runtime provider.
	// It is the only result for which callers may use the legacy GenGo path.
	ErrNotHandled = errors.New("runtime provider not configured")

	ErrRuntimeVendorUnsupported = errors.New("runtime providers do not support vendor mode")
	ErrRuntimeDisabled          = errors.New("runtime provider execution is disabled by XGO_RUNTIME=off")
	ErrRuntimeRecursive         = errors.New("recursive runtime provider invocation")
	ErrRuntimeArgvTooLarge      = errors.New("runtime provider argv and environment are too large")
)

type action = runtimeprotocol.Action

const (
	actionRun   = runtimeprotocol.ActionRun
	actionBuild = runtimeprotocol.ActionBuild
)

// TargetKind records the user-facing form of the resolved target.
type TargetKind int

const (
	TargetDirectory TargetKind = iota
	TargetFile
	TargetPackage
)

// ModuleRef and ResolvedModule share xgomod's canonical resolved identity.
type ModuleRef = xgomod.ModuleRef
type ResolvedModule = xgomod.ResolvedModule

type modMode string

const (
	modModeMod      modMode = "mod"
	modModeReadonly modMode = "readonly"
	modModeVendor   modMode = "vendor"
)

// GraphPolicy is the exact module/workspace policy shared by discovery,
// validation, and provider construction.
type GraphPolicy struct {
	GoCommand string
	GoWork    string
	ModMode   modMode
	ModFile   string
	Overlay   string
	// WorkDir anchors all Go graph operations on both sides of the wire;
	// provider execution itself still runs in ProjectDir.
	WorkDir string
}

// BuildPolicy is the runtime-safe subset of XGo/Go build flags.
type BuildPolicy struct {
	Verbose         bool
	Trace           bool
	KeepWork        bool
	TrimPath        bool
	DisableBuildVCS bool
}

// Runtime is the immutable discovery result passed to execution.
type Runtime struct {
	TargetKind       TargetKind
	OriginalTarget   string
	TargetImportPath string
	DefaultExecName  string
	ProjectDir       string
	ProjectFile      string
	ModuleRoot       string
	ProviderPackage  string
	Origin           ResolvedModule
	RequiredXGo      string
	Protocol         string
	ProjectExt       string
	ProjectFullExt   string
	PackDir          string
	PackIndex        string
	GoxMod           string
	GoxModSHA256     string
	Graph            GraphPolicy
}

// Streams are inherited by the provider without using protocol files or stdin.
type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func defaultStreams() Streams {
	return Streams{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}
}

// ProcessStatus preserves a normal exit code or an operating-system signal.
type ProcessStatus struct {
	Code     int
	Signal   os.Signal
	Signaled bool
}

func successStatus() ProcessStatus { return ProcessStatus{Code: 0} }
