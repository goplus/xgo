/*
 * Copyright (c) 2022 The XGo Authors (xgo.dev). All rights reserved.
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

package xgoenv

import (
	"github.com/goplus/xgo/env"
)

// XGo represents the XGo environment information.
type XGo struct {
	Version   string
	BuildDate string
	Root      string // XGOROOT
}

// Get returns the current XGo environment information.
func Get() *XGo {
	return &XGo{
		Version:   env.Version(),
		BuildDate: env.BuildDate(),
		Root:      env.XGOROOT(),
	}
}
