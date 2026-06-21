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

package typeargs

const (
	XGoPackage = true
)

// -----------------------------------------------------------------------------

// XGox_As has one non-inferable TypeParam (T, only in return position)
// and no inferable TypeParams.
func XGox_As[T any](src any) (T, error) {
	panic("not implemented")
}

// XGox_Convert has one non-inferable TypeParam (To, only in return position)
// and one inferable TypeParam (From, appears in the parameter list).
func XGox_Convert[To any, From any](src From) To {
	panic("not implemented")
}

// -----------------------------------------------------------------------------

type App struct {
}

func (p *App) initApp() {}

type iAppProto interface {
	initApp()
}

func XGot_App_XGox_OnCall[InT any, AppT iAppProto](a AppT, callback func(args *InT)) {
	panic("not implemented")
}

// -----------------------------------------------------------------------------
