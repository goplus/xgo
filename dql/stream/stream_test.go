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

package stream_test

import (
	"io"
	"testing"

	"github.com/goplus/xgo/dql/stream"
	_ "github.com/goplus/xgo/dql/stream/inline"
)

func TestBasic(t *testing.T) {
	f, err := stream.Open("inline:hello")
	if err != nil {
		t.Fatal("Open failed:", err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal("ioutil.ReadAll failed:", err)
	}
	if string(b) != "hello" {
		t.Fatal("unexpected data")
	}
}

func TestUnknownScheme(t *testing.T) {
	_, err := stream.Open("bad://foo")
	if err == nil || err.Error() != "dql/stream.Open bad://foo: unknown scheme" {
		t.Fatal("Open failed:", err)
	}
}

func TestOpenFile(t *testing.T) {
	_, err := stream.Open("/bin/not-exists/foo")
	if err == nil {
		t.Fatal("Open local file success?")
	}
}
