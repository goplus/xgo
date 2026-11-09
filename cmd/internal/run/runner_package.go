package run

import (
	"errors"
	"path/filepath"

	"github.com/goplus/mod/modcache"
	"github.com/goplus/mod/modfetch"
	"github.com/goplus/mod/xgomod"
)

func downloadPackageDir(pkgPath, version string) (string, error) {
	spec := packageRef(pkgPath, version)
	modVer, relPath, err := modfetch.GetPkg(spec, "")
	if err != nil {
		return "", err
	}
	modDir, err := modcache.Path(modVer)
	if err != nil {
		return "", err
	}
	directory := modDir
	if relPath != "" {
		directory = filepath.Join(modDir, relPath)
	}
	return filepath.Abs(directory)
}

func lookupPackageDir(workDir, pkgPath string) (string, error) {
	mod, err := xgomod.Load(workDir)
	if err = ignoreMissing(err); err != nil {
		return "", err
	}
	if mod == nil {
		return "", nil
	}

	pkg, err := mod.Lookup(pkgPath)
	if err = ignoreMissing(err); err != nil {
		return "", err
	}
	if pkg == nil {
		return "", nil
	}

	directory, err := filepath.Abs(pkg.Dir)
	if err != nil {
		return "", err
	}
	return directory, nil
}

func packageRef(pkgPath, version string) string {
	if version == "" {
		version = "latest"
	}
	return pkgPath + "@" + version
}

func ignoreMissing(err error) error {
	if err == nil || xgomod.IsNotFound(err) {
		return nil
	}
	var missing *xgomod.MissingError
	if errors.As(err, &missing) {
		return nil
	}
	return err
}
