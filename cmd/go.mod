module github.com/goplus/xgo/cmd

go 1.27.0

require (
	github.com/goplus/cobra v1.10.8 //xgo:class
	github.com/goplus/gogen v1.26.6
	github.com/goplus/mod v0.22.1
	github.com/goplus/xgo v0.0.0-00010101000000-000000000000
	github.com/qiniu/x v1.19.3
)

require (
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	golang.org/x/mod v0.40.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

retract v1.1.12

replace github.com/goplus/xgo => ../
