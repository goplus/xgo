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
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSplitQuotedFields(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "whole fields may be quoted",
			in:   `-buildvcs=false '-overlay=a b.json' "-modfile=x.mod" -trimpath`,
			want: []string{"-buildvcs=false", "-overlay=a b.json", "-modfile=x.mod", "-trimpath"},
		},
		{
			name: "backslashes are literal",
			in:   `"-modfile=C:\work\alt.mod" -overlay=C:\work\overlay.json -x=foo\`,
			want: []string{`-modfile=C:\work\alt.mod`, `-overlay=C:\work\overlay.json`, `-x=foo\`},
		},
		{
			name: "quotes inside fields are literal",
			in:   `-overlay="a b.json" "-modfile=x.mod"`,
			want: []string{`-overlay="a`, `b.json"`, "-modfile=x.mod"},
		},
		{
			name: "unterminated interior quote is literal",
			in:   `-x="unterminated`,
			want: []string{`-x="unterminated`},
		},
		{
			name: "empty quoted field",
			in:   `'' ""`,
			want: []string{"", ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitQuotedFields(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitQuotedFields(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
	for _, in := range []string{`"unterminated`, `'unterminated`} {
		if _, err := splitQuotedFields(in); err == nil {
			t.Fatalf("splitQuotedFields(%q) succeeded", in)
		}
	}
}

func TestParseDriverFlags(t *testing.T) {
	dir := t.TempDir()
	got, err := parseDriverFlags(dir, "/usr/bin/go", "off",
		`-buildvcs=false -trimpath=true -mod=readonly`,
		[]string{"-v=true", "-x=false", "-work=true", "-trimpath=false", "-buildvcs=auto", "-mod=mod", "-modfile=alt.mod"})
	if err != nil {
		t.Fatal(err)
	}
	wantGraph := []string{
		"-mod=mod",
		"-modfile=" + filepath.Join(dir, "alt.mod"),
	}
	if !reflect.DeepEqual(got.graph.goFlags(), wantGraph) {
		t.Fatalf("graph flags = %#v, want %#v", got.graph.goFlags(), wantGraph)
	}
	if got.graph.ModMode != modModeMod || got.graph.ModFile != filepath.Join(dir, "alt.mod") || !got.build.Verbose || got.build.Trace || !got.build.KeepWork {
		t.Fatalf("unexpected policies: %#v", got)
	}
	if got.build.DisableBuildVCS || got.build.TrimPath {
		t.Fatalf("explicit defaults were not normalized: %#v", got.build)
	}
	if err := got.validateDriver(); err != nil {
		t.Fatal(err)
	}
}

func TestParseDriverFlagsDefersOverlayRejection(t *testing.T) {
	dir := t.TempDir()
	got, err := parseDriverFlags(dir, "/usr/bin/go", "off", `'-overlay=old overlay.json'`, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantGraph := []string{"-overlay=" + filepath.Join(dir, "old overlay.json")}
	if !reflect.DeepEqual(got.graph.goFlags(), wantGraph) {
		t.Fatalf("graph flags = %#v, want %#v", got.graph.goFlags(), wantGraph)
	}
	if err := got.validateDriver(); err == nil || !strings.Contains(err.Error(), "overlay") {
		t.Fatalf("validateDriver() = %v, want deferred overlay rejection", err)
	}
}

func TestParseDriverFlagsAcceptsDoubleDashForms(t *testing.T) {
	dir := t.TempDir()
	got, err := parseDriverFlags(dir, "/usr/bin/go", "off", `--mod=readonly --trimpath`, []string{"--mod=mod", "--buildvcs=false"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"-mod=mod"}; !reflect.DeepEqual(got.graph.goFlags(), want) {
		t.Fatalf("graph flags = %#v, want %#v", got.graph.goFlags(), want)
	}
	if !got.build.TrimPath || !got.build.DisableBuildVCS {
		t.Fatalf("build flags = %#v", got.build)
	}
	if err := got.validateDriver(); err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"--", "---trimpath"} {
		got, err := parseDriverFlags(dir, "/usr/bin/go", "off", "", []string{flag})
		if err != nil {
			t.Fatal(err)
		}
		if err := got.validateDriver(); err == nil {
			t.Fatalf("invalid flag %q was accepted", flag)
		}
	}
}

func TestBuildPolicyFlagForms(t *testing.T) {
	policy := BuildPolicy{
		TrimPath: true,
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

func TestParseDriverFlagsRejected(t *testing.T) {
	for _, flag := range []string{"-n=true", "-tags=foo", "-buildmode=pie", "-buildvcs=true", "-trimpath=invalid"} {
		got, err := parseDriverFlags(t.TempDir(), "/usr/bin/go", "off", "", []string{flag})
		if err != nil {
			t.Fatalf("parseDriverFlags(%q): %v", flag, err)
		}
		if err := got.validateDriver(); err == nil || !strings.Contains(err.Error(), strings.Split(strings.TrimPrefix(flag, "-"), "=")[0]) {
			t.Fatalf("validateDriver(%q) = %v", flag, err)
		}
	}
}
