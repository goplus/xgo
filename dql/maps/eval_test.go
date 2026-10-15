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

package maps

import (
	"slices"
	"testing"
)

func TestEval(t *testing.T) {
	projectDoc := map[string]any{
		"assets": map[string]any{
			"users": []any{
				map[string]any{"name": "ken", "role": "admin"},
				map[string]any{"name": "jane", "role": "user"},
			},
			"sprites": map[string]any{
				"Hero": map[string]any{
					"index.json": map[string]any{
						"costumes": []any{
							map[string]any{
								"name": "idle",
								"frames": []any{
									map[string]any{"name": "f1"},
									map[string]any{"name": "f2"},
								},
							},
							map[string]any{
								"name": "run",
								"frames": []any{
									map[string]any{"name": "f1"},
								},
							},
						},
						"zorder": []any{
							map[string]any{"name": "front"},
							map[string]any{"name": ""},
							map[string]any{"other": "skip"},
						},
					},
				},
				"Enemy": map[string]any{
					"index.json": map[string]any{
						"costumes": []any{
							map[string]any{
								"name": "idle",
								"frames": []any{
									map[string]any{"name": "f1"},
								},
							},
						},
					},
				},
			},
		},
	}

	t.Run("WildcardChild", func(t *testing.T) {
		nodes, err := Eval(`assets.sprites.*`, projectDoc)
		if err != nil {
			t.Fatalf("Eval returned error: %v", err)
		}
		names := collectNames(nodes)
		slices.Sort(names)
		if want := []string{"Enemy", "Hero"}; !slices.Equal(names, want) {
			t.Fatalf("collectNames() = %v, want %v", names, want)
		}
	})

	t.Run("RelativeQuotedRoot", func(t *testing.T) {
		hero := childNode(t, projectDoc, "assets", "sprites", "Hero")
		nodes, err := Eval(`"index.json".costumes.*`, hero)
		if err != nil {
			t.Fatalf("Eval returned error: %v", err)
		}
		names := collectStringAttr(nodes, "name")
		if want := []string{"idle", "run"}; !slices.Equal(names, want) {
			t.Fatalf("collectStringAttr() = %v, want %v", names, want)
		}
	})

	t.Run("RelativeFrames", func(t *testing.T) {
		hero := childNode(t, projectDoc, "assets", "sprites", "Hero")
		costumes, err := Eval(`"index.json".costumes.*`, hero)
		if err != nil {
			t.Fatalf("Eval returned error: %v", err)
		}
		first := firstNode(t, costumes)
		frames, err := Eval(`frames.*`, first)
		if err != nil {
			t.Fatalf("Eval returned error: %v", err)
		}
		names := collectStringAttr(frames, "name")
		if want := []string{"f1", "f2"}; !slices.Equal(names, want) {
			t.Fatalf("collectStringAttr() = %v, want %v", names, want)
		}
	})

	t.Run("NameChild", func(t *testing.T) {
		hero := childNode(t, projectDoc, "assets", "sprites", "Hero")
		costumes, err := Eval(`"index.json".costumes.*`, hero)
		if err != nil {
			t.Fatalf("Eval returned error: %v", err)
		}
		first := firstNode(t, costumes)
		nameNodes, err := Eval(`name`, first)
		if err != nil {
			t.Fatalf("Eval returned error: %v", err)
		}
		node := firstNode(t, nameNodes)
		if node.Name != "name" {
			t.Fatalf("node.Name = %q, want %q", node.Name, "name")
		}
		if text, ok := node.Value.(string); !ok || text != "idle" {
			t.Fatalf("node.Value = %#v, want %q", node.Value, "idle")
		}
	})

	t.Run("FilterExpr", func(t *testing.T) {
		hero := childNode(t, projectDoc, "assets", "sprites", "Hero")
		nodes, err := Eval(`"index.json".zorder.*@($name != "")`, hero)
		if err != nil {
			t.Fatalf("Eval returned error: %v", err)
		}
		names := collectStringAttr(nodes, "name")
		if want := []string{"front"}; !slices.Equal(names, want) {
			t.Fatalf("collectStringAttr() = %v, want %v", names, want)
		}
	})

	t.Run("SelectByName", func(t *testing.T) {
		nodes, err := Eval(`assets.sprites.*@Hero`, projectDoc)
		if err != nil {
			t.Fatalf("Eval returned error: %v", err)
		}
		names := collectNames(nodes)
		if want := []string{"Hero"}; !slices.Equal(names, want) {
			t.Fatalf("collectNames() = %v, want %v", names, want)
		}
	})

	t.Run("Index", func(t *testing.T) {
		nodes, err := Eval(`assets.users[0]`, projectDoc)
		if err != nil {
			t.Fatalf("Eval returned error: %v", err)
		}
		names := collectStringAttr(nodes, "name")
		if want := []string{"ken"}; !slices.Equal(names, want) {
			t.Fatalf("collectStringAttr() = %v, want %v", names, want)
		}
	})

	t.Run("Single", func(t *testing.T) {
		nodes, err := Eval(`assets.users.*@($role == "admin")._single`, projectDoc)
		if err != nil {
			t.Fatalf("Eval returned error: %v", err)
		}
		if nodes.Err != nil {
			t.Fatalf("nodes.Err = %v, want nil", nodes.Err)
		}
		names := collectStringAttr(nodes, "name")
		if want := []string{"ken"}; !slices.Equal(names, want) {
			t.Fatalf("collectStringAttr() = %v, want %v", names, want)
		}
	})

	t.Run("MatchCall", func(t *testing.T) {
		nodes, err := Eval(`assets.users.*@match("k*", $name)`, projectDoc)
		if err != nil {
			t.Fatalf("Eval returned error: %v", err)
		}
		names := collectStringAttr(nodes, "name")
		if want := []string{"ken"}; !slices.Equal(names, want) {
			t.Fatalf("collectStringAttr() = %v, want %v", names, want)
		}
	})

	t.Run("StringMethodCall", func(t *testing.T) {
		nodes, err := Eval(`assets.users.*@($name.hasPrefix("k"))`, projectDoc)
		if err != nil {
			t.Fatalf("Eval returned error: %v", err)
		}
		names := collectStringAttr(nodes, "name")
		if want := []string{"ken"}; !slices.Equal(names, want) {
			t.Fatalf("collectStringAttr() = %v, want %v", names, want)
		}
	})

	t.Run("AnySelector", func(t *testing.T) {
		nodes, err := Eval(`assets.**.name`, projectDoc)
		if err != nil {
			t.Fatalf("Eval returned error: %v", err)
		}
		names := collectScalarStrings(nodes)
		slices.Sort(names)
		want := []string{"", "f1", "f1", "f1", "f2", "front", "idle", "idle", "jane", "ken", "run"}
		if !slices.Equal(names, want) {
			t.Fatalf("collectScalarStrings() = %v, want %v", names, want)
		}
	})
}

func collectNames(nodes NodeSet) []string {
	ret := make([]string, 0, 8)
	nodes.Data(func(node Node) bool {
		ret = append(ret, node.Name)
		return true
	})
	return ret
}

func collectStringAttr(nodes NodeSet, name string) []string {
	ret := make([]string, 0, 8)
	nodes.Data(func(node Node) bool {
		if children, ok := node.Value.(map[string]any); ok {
			if value, ok := children[name].(string); ok {
				ret = append(ret, value)
			}
		}
		return true
	})
	return ret
}

func collectScalarStrings(nodes NodeSet) []string {
	ret := make([]string, 0, 8)
	nodes.Data(func(node Node) bool {
		if value, ok := node.Value.(string); ok {
			ret = append(ret, value)
		}
		return true
	})
	return ret
}

func firstNode(t *testing.T, nodes NodeSet) Node {
	t.Helper()
	node, err := nodes.XGo_first()
	if err != nil {
		t.Fatalf("XGo_first() error = %v", err)
	}
	return node
}

func childNode(t *testing.T, root any, path ...string) Node {
	t.Helper()
	current := Source(root)
	for _, name := range path {
		current = current.XGo_Elem(name)
	}
	node, err := current.XGo_first()
	if err != nil {
		t.Fatalf("childNode(%v) error = %v", path, err)
	}
	return node
}
