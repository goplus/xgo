/*
 * Copyright (c) 2023 The XGo Authors (xgo.dev). All rights reserved.
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

package watcher

import (
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/goplus/mod/goxmod"
	"golang.org/x/mod/modfile"
)

// -----------------------------------------------------------------------------------------

type module struct {
	exts []string
}

func (p *module) ignore(fname string) bool {
	ext := path.Ext(fname)
	for _, v := range p.exts {
		if ext == v {
			return false
		}
	}
	return true
}

// -----------------------------------------------------------------------------------------

type none struct{}

type Changes struct {
	changed map[string]none
	mods    map[string]*module
	mutex   sync.Mutex
	cond    sync.Cond

	root string
}

func NewChanges(root string) *Changes {
	changed := make(map[string]none)
	mods := make(map[string]*module)
	root, _ = filepath.Abs(root)
	c := &Changes{changed: changed, mods: mods, root: root + "/"}
	c.cond.L = &c.mutex
	return c
}

func (p *Changes) doLookupMod(name string) *module {
	mod, ok := p.mods[name]
	if !ok {
		mod = new(module)
		mod.exts = make([]string, 0, 8)
		for _, ext := range classExts(p.root + name) {
			mod.exts = append(mod.exts, ext)
		}
		mod.exts = append(mod.exts, ".xgo", ".gop", ".go", ".gox")
		p.mods[name] = mod
		if debugMod {
			log.Println("Mod:", name, "Exts:", mod.exts)
		}
	}
	return mod
}

// classExts returns the classfile extensions declared by the module rooted at
// dir (through its gox.mod, legacy gop.mod, or //xgo:class requires). It is a
// best-effort read: a directory with no module, or an unreadable one, yields no
// extra extensions.
func classExts(dir string) []string {
	gomodData, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return nil
	}
	gomod, err := modfile.Parse("go.mod", gomodData, nil)
	if err != nil {
		return nil
	}
	gox, err := goxmod.LoadModfile(gomod, filepath.Join(dir, "gox.mod"))
	if err != nil {
		return nil
	}
	var exts []string
	for _, c := range gox.Projects {
		exts = append(exts, c.Ext)
		for _, w := range c.Works {
			if w.Ext != c.Ext {
				exts = append(exts, w.Ext)
			}
		}
	}
	return exts
}

func (p *Changes) lookupMod(name string) *module {
	name = strings.TrimSuffix(name, "/")
	p.mutex.Lock()
	mod := p.doLookupMod(name)
	p.mutex.Unlock()
	return mod
}

func (p *Changes) deleteMod(dir string) {
	p.mutex.Lock()
	delete(p.mods, dir)
	p.mutex.Unlock()
}

func (p *Changes) Fetch(fullPath bool) (dir string) {
	p.mutex.Lock()
	for len(p.changed) == 0 {
		p.cond.Wait()
	}
	for dir = range p.changed {
		delete(p.changed, dir)
		break
	}
	p.mutex.Unlock()
	if fullPath {
		dir = p.root + dir
	}
	return
}

func (p *Changes) Ignore(name string, isDir bool) bool {
	dir, fname := path.Split(name)
	if strings.HasPrefix(fname, "_") {
		return true
	}
	return !isDir && (isHiddenTemp(fname) || isAutogen(fname) || p.lookupMod(dir).ignore(fname))
}

func (p *Changes) FileChanged(name string) {
	dir := path.Dir(name)
	p.mutex.Lock()
	n := len(p.changed)
	p.changed[dir] = none{}
	p.mutex.Unlock()
	if n == 0 {
		p.cond.Broadcast()
	}
}

func (p *Changes) EntryDeleted(name string, isDir bool) {
	if isDir {
		p.deleteMod(name)
	} else {
		p.FileChanged(name)
	}
}

func (p *Changes) DirAdded(name string) {
	dir := p.root + name
	filepath.WalkDir(dir, func(entry string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		entry, _ = filepath.Rel(dir, entry)
		entry = filepath.ToSlash(entry)
		if !p.Ignore(entry, false) {
			p.FileChanged(entry)
		}
		return nil
	})
}

// pattern: xgo_autogen*.go
func isAutogen(fname string) bool {
	return strings.HasPrefix(fname, "xgo_autogen") && strings.HasSuffix(fname, ".go")
}

// pattern: .* or *~
func isHiddenTemp(fname string) bool {
	return strings.HasPrefix(fname, ".") || strings.HasSuffix(fname, "~")
}

// -----------------------------------------------------------------------------------------
