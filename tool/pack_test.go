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

package tool

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeJSON is a test helper that writes obj as JSON to dir/filename.
func writeJSON(t *testing.T, dir, filename string, obj any) {
	t.Helper()
	data, err := json.MarshalIndent(obj, "", "\t")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, filename), data, 0644); err != nil {
		t.Fatal(err)
	}
}

// readJSON is a test helper that reads and parses a JSON file.
func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatal(err)
	}
	return obj
}

// setupSPXLayout creates the example SPX layout from the proposal:
//
//	assets/
//	  index.json
//	  sprites/Cat/index.json
//	  sprites/Balloon/index.json
//	  sounds/bgm/index.json
func setupSPXLayout(t *testing.T, root string) {
	t.Helper()

	assets := filepath.Join(root, "assets")
	dirs := []string{
		assets,
		filepath.Join(assets, "sprites", "Cat"),
		filepath.Join(assets, "sprites", "Balloon"),
		filepath.Join(assets, "sounds", "bgm"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	writeJSON(t, assets, "index.json", map[string]any{
		"zorder": []string{"Cat", "Balloon"},
		"map":    map[string]any{"width": 480, "height": 360},
	})
	writeJSON(t, filepath.Join(assets, "sprites", "Cat"), "index.json", map[string]any{
		"x": 0, "y": 0, "size": 100,
		"costumes": []map[string]any{{"name": "default", "path": "cat.png"}},
	})
	writeJSON(t, filepath.Join(assets, "sprites", "Balloon"), "index.json", map[string]any{
		"x": 100, "y": 50, "size": 80,
	})
	writeJSON(t, filepath.Join(assets, "sounds", "bgm"), "index.json", map[string]any{
		"path":   "bgm.mp3",
		"volume": 80,
	})
}

func TestPackBasicSPXLayout(t *testing.T) {
	root := t.TempDir()
	setupSPXLayout(t, root)

	if err := Pack(root, 0); err != nil {
		t.Fatal("Pack failed:", err)
	}

	packFile := filepath.Join(root, "assets", "index_pack.json")
	if _, err := os.Stat(packFile); os.IsNotExist(err) {
		t.Fatal("index_pack.json not created")
	}

	obj := readJSON(t, packFile)

	// Verify root fields are preserved
	if _, ok := obj["zorder"]; !ok {
		t.Error("missing root field 'zorder'")
	}
	if _, ok := obj["map"]; !ok {
		t.Error("missing root field 'map'")
	}

	// Verify sprites are merged
	sprites, ok := obj["sprites"].(map[string]any)
	if !ok {
		t.Fatal("missing or invalid 'sprites' key")
	}
	if _, ok := sprites["Cat"]; !ok {
		t.Error("missing sprites.Cat")
	}
	if _, ok := sprites["Balloon"]; !ok {
		t.Error("missing sprites.Balloon")
	}

	// Verify Cat contents
	cat, ok := sprites["Cat"].(map[string]any)
	if !ok {
		t.Fatal("sprites.Cat is not an object")
	}
	if cat["size"] != float64(100) {
		t.Errorf("sprites.Cat.size = %v, want 100", cat["size"])
	}

	// Verify sounds are merged
	sounds, ok := obj["sounds"].(map[string]any)
	if !ok {
		t.Fatal("missing or invalid 'sounds' key")
	}
	bgm, ok := sounds["bgm"].(map[string]any)
	if !ok {
		t.Fatal("sounds.bgm is not an object")
	}
	if bgm["volume"] != float64(80) {
		t.Errorf("sounds.bgm.volume = %v, want 80", bgm["volume"])
	}
}

func TestPackTestMode(t *testing.T) {
	root := t.TempDir()
	setupSPXLayout(t, root)

	// Test mode should fail when no pack file exists
	if err := Pack(root, PackFlagTest); err == nil {
		t.Fatal("expected error in test mode with no pack file, got nil")
	}

	// Generate the pack file
	if err := Pack(root, 0); err != nil {
		t.Fatal("Pack failed:", err)
	}

	// Test mode should succeed now
	if err := Pack(root, PackFlagTest); err != nil {
		t.Fatal("test mode failed after packing:", err)
	}

	// Tamper with pack file
	packFile := filepath.Join(root, "assets", "index_pack.json")
	if err := os.WriteFile(packFile, []byte(`{"tampered": true}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Test mode should fail after tampering
	if err := Pack(root, PackFlagTest); err == nil {
		t.Fatal("expected error in test mode after tampering, got nil")
	}
}

func TestPackNoConfigFiles(t *testing.T) {
	root := t.TempDir()

	// Should return nil (warning only) when no config files found
	if err := Pack(root, 0); err != nil {
		t.Fatal("expected nil error for empty directory, got:", err)
	}
}

func TestPackMultipleConfigFormats(t *testing.T) {
	root := t.TempDir()

	// Create both index.json and index.yaml in the same directory
	writeJSON(t, root, "index.json", map[string]any{"a": 1})
	if err := os.WriteFile(filepath.Join(root, "index.yaml"), []byte("a: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := Pack(root, 0)
	if err == nil {
		t.Fatal("expected error for multiple config formats, got nil")
	}
	want := "pack: walking directory tree: pack: directory . contains multiple config files: index.json and index.yaml"
	if err.Error() != want {
		t.Fatalf("err.Error() = %q, want %q", err.Error(), want)
	}
}

func TestPackCollisionDetection(t *testing.T) {
	root := t.TempDir()

	// Root config already has a "sprites" key with a non-object value
	if err := os.MkdirAll(filepath.Join(root, "sprites", "Cat"), 0755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, root, "index.json", map[string]any{
		"sprites": "this-is-not-an-object",
	})
	writeJSON(t, filepath.Join(root, "sprites", "Cat"), "index.json", map[string]any{
		"x": 0,
	})

	err := Pack(root, 0)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	want := `pack: collision: key "sprites" at path "sprites" is not an object (introduced by directory structure, conflicts with sprites/Cat/index.json)`
	if err.Error() != want {
		t.Fatalf("err.Error() = %q, want %q", err.Error(), want)
	}
}

func TestPackKeyCollisionAtLeaf(t *testing.T) {
	root := t.TempDir()

	// Root config has a key "items" that is an object, and a child
	// directory "items/foo" would try to nest under it.
	// But root.items already has a key "foo".
	if err := os.MkdirAll(filepath.Join(root, "items", "foo"), 0755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, root, "index.json", map[string]any{
		"items": map[string]any{
			"foo": "already-here",
		},
	})
	writeJSON(t, filepath.Join(root, "items", "foo"), "index.json", map[string]any{
		"value": 42,
	})

	err := Pack(root, 0)
	if err == nil {
		t.Fatal("expected collision error at leaf, got nil")
	}
	want := `pack: collision: key "foo" already exists at path "items/foo" (introduced by items/foo/index.json)`
	if err.Error() != want {
		t.Fatalf("err.Error() = %q, want %q", err.Error(), want)
	}
}

func TestPackDeterminism(t *testing.T) {
	root := t.TempDir()
	setupSPXLayout(t, root)

	if err := Pack(root, 0); err != nil {
		t.Fatal(err)
	}
	packFile := filepath.Join(root, "assets", "index_pack.json")
	first, err := os.ReadFile(packFile)
	if err != nil {
		t.Fatal(err)
	}

	// Remove and regenerate
	os.Remove(packFile)
	if err := Pack(root, 0); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(packFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != string(second) {
		t.Error("Pack output is not deterministic")
	}
}

func TestPackMultipleRoots(t *testing.T) {
	root := t.TempDir()

	// Create two independent pack roots
	game1 := filepath.Join(root, "game1")
	game2 := filepath.Join(root, "game2")
	for _, dir := range []string{
		game1,
		filepath.Join(game1, "sprites", "Player"),
		game2,
		filepath.Join(game2, "sounds", "fx"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	writeJSON(t, game1, "index.json", map[string]any{"title": "Game1"})
	writeJSON(t, filepath.Join(game1, "sprites", "Player"), "index.json", map[string]any{"hp": 100})
	writeJSON(t, game2, "index.json", map[string]any{"title": "Game2"})
	writeJSON(t, filepath.Join(game2, "sounds", "fx"), "index.json", map[string]any{"volume": 50})

	if err := Pack(root, 0); err != nil {
		t.Fatal("Pack failed:", err)
	}

	// Both roots should have pack files
	for _, dir := range []string{game1, game2} {
		if _, err := os.Stat(filepath.Join(dir, "index_pack.json")); os.IsNotExist(err) {
			t.Errorf("index_pack.json not created in %s", dir)
		}
	}

	obj1 := readJSON(t, filepath.Join(game1, "index_pack.json"))
	sprites, ok := obj1["sprites"].(map[string]any)
	if !ok {
		t.Fatal("game1: missing sprites")
	}
	player, ok := sprites["Player"].(map[string]any)
	if !ok {
		t.Fatal("game1: missing sprites.Player")
	}
	if player["hp"] != float64(100) {
		t.Errorf("game1: sprites.Player.hp = %v, want 100", player["hp"])
	}

	obj2 := readJSON(t, filepath.Join(game2, "index_pack.json"))
	sounds, ok := obj2["sounds"].(map[string]any)
	if !ok {
		t.Fatal("game2: missing sounds")
	}
	fx, ok := sounds["fx"].(map[string]any)
	if !ok {
		t.Fatal("game2: missing sounds.fx")
	}
	if fx["volume"] != float64(50) {
		t.Errorf("game2: sounds.fx.volume = %v, want 50", fx["volume"])
	}
}

func TestPackDeeplyNested(t *testing.T) {
	root := t.TempDir()

	// Three levels of nesting: root -> a -> b -> c
	dirs := []string{
		root,
		filepath.Join(root, "a", "b", "c"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	writeJSON(t, root, "index.json", map[string]any{"root": true})
	writeJSON(t, filepath.Join(root, "a", "b", "c"), "index.json", map[string]any{"deep": true})

	if err := Pack(root, 0); err != nil {
		t.Fatal("Pack failed:", err)
	}

	obj := readJSON(t, filepath.Join(root, "index_pack.json"))

	// Verify nested structure: root.a.b.c
	a, ok := obj["a"].(map[string]any)
	if !ok {
		t.Fatal("missing key 'a'")
	}
	b, ok := a["b"].(map[string]any)
	if !ok {
		t.Fatal("missing key 'a.b'")
	}
	c, ok := b["c"].(map[string]any)
	if !ok {
		t.Fatal("missing key 'a.b.c'")
	}
	if c["deep"] != true {
		t.Errorf("a.b.c.deep = %v, want true", c["deep"])
	}
}

func TestPackSkipsSubdirsWithoutConfig(t *testing.T) {
	root := t.TempDir()

	// Create a root with a subdirectory that has no index.json (assets only)
	if err := os.MkdirAll(filepath.Join(root, "images"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "images", "logo.png"), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sprites", "Cat"), 0755); err != nil {
		t.Fatal(err)
	}

	writeJSON(t, root, "index.json", map[string]any{"title": "test"})
	writeJSON(t, filepath.Join(root, "sprites", "Cat"), "index.json", map[string]any{"x": 0})

	if err := Pack(root, 0); err != nil {
		t.Fatal("Pack failed:", err)
	}

	obj := readJSON(t, filepath.Join(root, "index_pack.json"))

	// "images" should not appear in packed output
	if _, ok := obj["images"]; ok {
		t.Error("unexpected key 'images' in packed output")
	}

	// "sprites.Cat" should be present
	sprites, ok := obj["sprites"].(map[string]any)
	if !ok {
		t.Fatal("missing 'sprites'")
	}
	if _, ok := sprites["Cat"]; !ok {
		t.Error("missing 'sprites.Cat'")
	}
}

func TestPackYAML(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "items", "sword"), 0755); err != nil {
		t.Fatal(err)
	}

	os.WriteFile(filepath.Join(root, "index.yaml"), []byte("title: game\n"), 0644)
	os.WriteFile(filepath.Join(root, "items", "sword", "index.yaml"), []byte("damage: 10\n"), 0644)

	if err := Pack(root, 0); err != nil {
		t.Fatal("Pack failed:", err)
	}

	packFile := filepath.Join(root, "index_pack.yaml")
	if _, err := os.Stat(packFile); os.IsNotExist(err) {
		t.Fatal("index_pack.yaml not created")
	}

	data, err := os.ReadFile(packFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if len(content) == 0 {
		t.Fatal("index_pack.yaml is empty")
	}
}

func TestPackFormatMismatch(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "items", "sword"), 0755); err != nil {
		t.Fatal(err)
	}

	// Root uses JSON, child uses YAML
	writeJSON(t, root, "index.json", map[string]any{"title": "game"})
	os.WriteFile(filepath.Join(root, "items", "sword", "index.yaml"), []byte("damage: 10\n"), 0644)

	err := Pack(root, 0)
	if err == nil {
		t.Fatal("expected format mismatch error, got nil")
	}
	want := "pack: format mismatch: items/sword/index.yaml uses .yaml but pack root index.json uses .json"
	if err.Error() != want {
		t.Fatalf("err.Error() = %q, want %q", err.Error(), want)
	}
}
