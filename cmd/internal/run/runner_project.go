package run

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goplus/mod/modcache"
	"github.com/goplus/mod/modfetch"
	"github.com/goplus/mod/modfile"
	"github.com/goplus/mod/xgomod"
	"github.com/goplus/xgo/x/xgoprojs"
)

func resolveProjectDir(proj xgoprojs.Proj, workDir string) (string, error) {
	switch v := proj.(type) {
	case *xgoprojs.DirProj:
		return absolutePath(workDir, v.Dir)
	case *xgoprojs.FilesProj:
		return resolveFilesProjectDir(workDir, v.Files)
	case *xgoprojs.PkgPathProj:
		return resolveProjectPackageDir(workDir, v.Path)
	default:
		return "", fmt.Errorf("unsupported project type %T", proj)
	}
}

func resolveFilesProjectDir(workDir string, files []string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files in project")
	}
	return absolutePath(workDir, filepath.Dir(files[0]))
}

func absolutePath(workDir, target string) (string, error) {
	if filepath.IsAbs(target) {
		return filepath.Clean(target), nil
	}
	if workDir == "" {
		workDir = "."
	}
	return filepath.Abs(filepath.Join(workDir, target))
}

func resolveProjectPackageDir(workDir, pkgPath string) (string, error) {
	if strings.HasSuffix(pkgPath, "/...") {
		return "", fmt.Errorf("project path %q cannot use /... with command runner", pkgPath)
	}
	pkgPath, version := splitPackageSpec(pkgPath)
	workDir = normalizeWorkDir(workDir)

	if dir, ok, err := resolveLocalPackageDir(workDir, pkgPath); err != nil {
		return "", err
	} else if ok {
		return dir, nil
	}
	return resolveDownloadedPackageDir(pkgPath, version)
}

func normalizeWorkDir(workDir string) string {
	if workDir == "" {
		return "."
	}
	return workDir
}

func resolveLocalPackageDir(workDir, pkgPath string) (string, bool, error) {
	pkg, err := lookupModulePackage(workDir, pkgPath)
	if err != nil {
		return "", false, err
	}
	if pkg == nil {
		return "", false, nil
	}
	return pkg.Dir, true, nil
}

func resolveDownloadedPackageDir(pkgPath, version string) (string, error) {
	spec := packageSpec(pkgPath, version)
	modVer, relPath, err := modfetch.GetPkg(spec, "")
	if err != nil {
		return "", err
	}
	modDir, err := modcache.Path(modVer)
	if err != nil {
		return "", err
	}
	dir := modDir
	if relPath != "" {
		dir = filepath.Join(modDir, relPath)
	}
	return filepath.Abs(dir)
}

func readCommandRunner(projectDir string) (*modfile.Runner, error) {
	return readRunnerFromGopMod(projectDir)
}

func readRunnerFromGopMod(projectDir string) (*modfile.Runner, error) {
	gopModPath, data, err := readProjectGopMod(projectDir)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	parsed, err := modfile.ParseLax(gopModPath, data, nil)
	if err != nil {
		return nil, err
	}
	if len(parsed.Projects) == 0 {
		return nil, nil
	}
	runner := parsed.Projects[0].Runner
	if err := validateRunnerSpec(runner); err != nil {
		return nil, err
	}
	return runner, nil
}

func validateRunnerSpec(runner *modfile.Runner) error {
	if runner == nil {
		return nil
	}
	if strings.HasSuffix(runner.Path, "/...") {
		return fmt.Errorf("runner path %q cannot use /... pattern", runner.Path)
	}
	if strings.Contains(runner.Path, "@") {
		basePath, _ := splitPackageSpec(runner.Path)
		return fmt.Errorf("runner path %q must not include @version; use `runner %s <version>`", runner.Path, basePath)
	}
	return nil
}

func readProjectGopMod(projectDir string) (string, []byte, error) {
	gopModPath := filepath.Join(projectDir, "gop.mod")
	data, err := os.ReadFile(gopModPath)
	if err != nil {
		if os.IsNotExist(err) {
			return gopModPath, nil, nil
		}
		return "", nil, err
	}
	return gopModPath, data, nil
}

func lookupModulePackage(workDir, pkgPath string) (*xgomod.Package, error) {
	mod, err := xgomod.Load(workDir)
	if err != nil {
		return nil, nil
	}
	pkg, err := mod.Lookup(pkgPath)
	if err != nil {
		return nil, nil
	}
	dir, err := filepath.Abs(pkg.Dir)
	if err != nil {
		return nil, err
	}
	pkg.Dir = dir
	return pkg, nil
}

func packageSpec(pkgPath, version string) string {
	if version == "" {
		version = "latest"
	}
	return pkgPath + "@" + version
}

func splitPackageSpec(pkgPath string) (string, string) {
	if pos := strings.IndexByte(pkgPath, '@'); pos > 0 {
		return pkgPath[:pos], pkgPath[pos+1:]
	}
	return pkgPath, ""
}
