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

package cached

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"

	"github.com/goplus/xgo/dql/stream"
	"github.com/goplus/xgo/dql/stream/http"
)

// -------------------------------------------------------------------------------------

var (
	cacheDir = getCacheDir()
)

func getCacheDir() string {
	root, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}
	dir := root + "/dql/http/"
	os.MkdirAll(dir, 0755)
	return dir
}

// -------------------------------------------------------------------------------------

// TODO(xsw): add checksum to cache file
func WriteCache(cacheFile string, url string) (err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	f, err := os.Create(cacheFile)
	if err != nil {
		return
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return
}

func ReadCache(cacheFile string, fi fs.FileInfo) (ret io.ReadCloser, err error) {
	return os.Open(cacheFile)
}

// -------------------------------------------------------------------------------------

// Open opens a http file object.
func Open(url_ string) (ret io.ReadCloser, err error) {
	u, err := url.Parse(url_)
	if err != nil {
		return
	}
	fname := path.Base(u.Path)
	ext := path.Ext(fname)
	hash := md5.Sum([]byte(url_))
	hashstr := base64.RawURLEncoding.EncodeToString(hash[:])
	fname = fmt.Sprintf("%s-%s%s", fname[:len(fname)-len(ext)], hashstr, ext)
	file := cacheDir + fname
	if fi, e := os.Stat(file); e == nil {
		if ret, err = ReadCache(file, fi); err == nil { // cache hit
			return
		}
	}
	if err = WriteCache(file, url_); err != nil {
		return // write cache failed
	}
	return ReadCache(file, nil)
}

func init() {
	stream.Register("http", Open)
	stream.Register("https", Open)
}

// -------------------------------------------------------------------------------------
