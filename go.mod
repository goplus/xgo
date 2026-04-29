module github.com/goplus/xgo

go 1.24.0

toolchain go1.24.2

require (
	github.com/fsnotify/fsnotify v1.9.0
	github.com/goccy/go-yaml v1.19.2
	github.com/goplus/cobra v1.9.16 //xgo:class
	github.com/goplus/gogen v1.23.0-pre.3.0.20260414234848-6641c10c9d6f
	github.com/goplus/lib v0.3.1
	github.com/goplus/mod v0.20.2
	github.com/qiniu/x v1.17.0
	golang.org/x/net v0.50.0
)

require (
	golang.org/x/mod v0.20.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

retract v1.1.12
