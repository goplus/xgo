/*
 * Copyright (c) 2021 The XGo Authors (xgo.dev). All rights reserved.
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
	"os"
	"os/exec"

	"github.com/goplus/mod/env"
	"github.com/goplus/mod/xgomod"
	"github.com/qiniu/x/errors"
)

func Tidy(dir string, xgo *env.XGo) (err error) {
	modObj, err := xgomod.Load(dir)
	if err != nil {
		return errors.NewWith(err, `xgomod.Load(dir, mod.GopModOnly)`, -2, "xgomod.Load", dir)
	}

	modRoot := modObj.Root()
	/*
		depMods, err := GenDepMods(modObj, modRoot, true)
		if err != nil {
			return errors.NewWith(err, `GenDepMods(modObj, modRoot, true)`, -2, "tool.GenDepMods", modObj, modRoot, true)
		}

		old := modObj.DepMods()
		for modPath := range old {
			if _, ok := depMods[modPath]; !ok { // removed
				modObj.DropRequire(modPath)
			}
		}
		for modPath := range depMods {
			if _, ok := old[modPath]; !ok { // added
				if newMod, e := modfetch.Get(modPath); e != nil {
					return errors.NewWith(e, `modfetch.Get(modPath)`, -1, "modfetch.Get", modPath)
				} else {
					modObj.AddRequire(newMod.Path, newMod.Version)
				}
			}
		}

		modObj.Cleanup()
		err = modObj.Save()
		if err != nil {
			return errors.NewWith(err, `modObj.Save()`, -2, "(*xgomod.Module).Save")
		}
	*/
	conf := &Config{XGo: xgo}
	err = genGoDir(modRoot, conf, true, true, 0)
	if err != nil {
		return errors.NewWith(err, `genGoDir(modRoot, conf, true, true)`, -2, "tool.genGoDir", modRoot, conf, true, true)
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = modRoot
	err = cmd.Run()
	if err != nil {
		err = errors.NewWith(err, `cmd.Run()`, -2, "(*exec.Cmd).Run")
	}
	return
}
