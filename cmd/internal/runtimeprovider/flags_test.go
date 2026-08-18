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
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSplitQuotedFields(t *testing.T) {
	got, err := splitQuotedFields(`-buildvcs=false '-overlay=a b.json' "-modfile=x.mod" -trimpath`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-buildvcs=false", "-overlay=a b.json", "-modfile=x.mod", "-trimpath"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitQuotedFields() = %#v, want %#v", got, want)
	}
	for _, in := range []string{`"unterminated`, `-x=foo\`} {
		if _, err := splitQuotedFields(in); err == nil {
			t.Fatalf("splitQuotedFields(%q) succeeded", in)
		}
	}
}

func TestJoinQuotedFieldsRoundTrip(t *testing.T) {
	want := []string{
		"-modfile=/workspace/with space/runtime.mod",
		`-overlay=C:\\work tree\\overlay.json`,
		`-overlay=/tmp/a\"quoted\".json`,
	}
	encoded := joinQuotedFields(want)
	got, err := splitQuotedFields(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitQuotedFields(joinQuotedFields()) = %#v, want %#v (encoded %q)", got, want, encoded)
	}
}

func TestParseRuntimeFlags(t *testing.T) {
	dir := t.TempDir()
	got, err := parseRuntimeFlags(dir, "/usr/bin/go", "off",
		`-buildvcs=false -mod=readonly -overlay='old overlay.json'`,
		[]string{"-v=true", "-x=false", "-work=true", "-trimpath=true", "-mod=mod", "-modfile=alt.mod"})
	if err != nil {
		t.Fatal(err)
	}
	wantGraph := []string{
		"-overlay=" + filepath.Join(dir, "old overlay.json"),
		"-mod=mod",
		"-modfile=" + filepath.Join(dir, "alt.mod"),
	}
	if !reflect.DeepEqual(got.graph.Flags, wantGraph) {
		t.Fatalf("graph flags = %#v, want %#v", got.graph.Flags, wantGraph)
	}
	if got.graph.ModMode != "mod" || !got.build.Verbose || got.build.Trace || !got.build.KeepWork {
		t.Fatalf("unexpected policies: %#v", got)
	}
	if want := []string{"-buildvcs=false", "-trimpath=true"}; !reflect.DeepEqual(got.build.Flags, want) {
		t.Fatalf("build flags = %#v, want %#v", got.build.Flags, want)
	}
	if err := got.validateRuntime(); err != nil {
		t.Fatal(err)
	}
}

func TestBuildPolicyFlagForms(t *testing.T) {
	policy := BuildPolicy{
		Flags:    []string{"-trimpath=true"},
		Verbose:  true,
		Trace:    true,
		KeepWork: true,
	}
	wantGo := []string{"-v", "-x", "-work", "-trimpath=true"}
	wantProtocol := []string{"-v=true", "-x=true", "-work=true", "-trimpath=true"}
	if got := policy.goBuildFlags(); !reflect.DeepEqual(got, wantGo) {
		t.Fatalf("go build flags = %#v, want %#v", got, wantGo)
	}
	if got := policy.protocolFlags(); !reflect.DeepEqual(got, wantProtocol) {
		t.Fatalf("protocol flags = %#v, want %#v", got, wantProtocol)
	}
}

func TestParseRuntimeFlagsRejected(t *testing.T) {
	for _, flag := range []string{"-n=true", "-tags=foo", "-buildmode=pie", "-buildvcs=true", "-trimpath=false"} {
		got, err := parseRuntimeFlags(t.TempDir(), "/usr/bin/go", "off", "", []string{flag})
		if err != nil {
			t.Fatalf("parseRuntimeFlags(%q): %v", flag, err)
		}
		if err := got.validateRuntime(); err == nil || !strings.Contains(err.Error(), strings.Split(strings.TrimPrefix(flag, "-"), "=")[0]) {
			t.Fatalf("validateRuntime(%q) = %v", flag, err)
		}
	}
}
