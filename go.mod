module github.com/goplus/xgo

go 1.24.0

toolchain go1.24.2

require (
	github.com/fsnotify/fsnotify v1.10.1
	github.com/goccy/go-yaml v1.19.2
	github.com/goplus/cobra v1.10.7 //xgo:class
	github.com/goplus/gogen v1.23.4-0.20260623084408-4e0857017abd
	github.com/goplus/lib v0.3.1
	github.com/goplus/mod v0.21.0
	github.com/qiniu/x v1.18.0
	golang.org/x/net v0.50.0
)

require (
	golang.org/x/mod v0.20.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

retract v1.1.12
