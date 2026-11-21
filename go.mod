module github.com/goplus/xgo

go 1.26.0

require (
	github.com/fsnotify/fsnotify v1.10.1
	github.com/goccy/go-yaml v1.19.2
	github.com/goplus/gogen v1.26.7
	github.com/goplus/lib v0.7.2
	github.com/goplus/mod v0.25.1
	github.com/qiniu/x v1.19.3
	golang.org/x/mod v0.40.0
	golang.org/x/net v0.62.0
)

require golang.org/x/sys v0.49.0 // indirect

retract (
	v1.7.6
	v1.1.12
)
