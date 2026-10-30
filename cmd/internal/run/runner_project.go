package run

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/goplus/mod/modfile"
	"github.com/goplus/xgo/x/xgoprojs"
	"golang.org/x/mod/module"
)

func resolveProjectDir(proj xgoprojs.Proj, workDir string) (string, error) {
	switch v := proj.(type) {
	case *xgoprojs.DirProj:
		return resolvePath(workDir, v.Dir)
	case *xgoprojs.FilesProj:
		return resolveFilesProjectDir(workDir, v.Files)
	case *xgoprojs.PkgPathProj:
		return resolvePackageProjectDir(workDir, v.Path)
	default:
		return "", fmt.Errorf("unsupported project type %T", proj)
	}
}

func resolveFilesProjectDir(workDir string, files []string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files in project")
	}
	return resolvePath(workDir, filepath.Dir(files[0]))
}

func resolvePath(workDir, target string) (string, error) {
	if filepath.IsAbs(target) {
		return filepath.Clean(target), nil
	}
	if workDir == "" {
		workDir = "."
	}
	return filepath.Abs(filepath.Join(workDir, target))
}

func resolvePackageProjectDir(workDir, pkgPath string) (string, error) {
	pkgPath, version, _ := strings.Cut(pkgPath, "@")
	if workDir == "" {
		workDir = "."
	}
	if packageDirectory, err := lookupPackageDir(workDir, pkgPath); err != nil {
		return "", err
	} else if packageDirectory != "" {
		return packageDirectory, nil
	}
	return downloadPackageDir(pkgPath, version)
}

func loadProjectRunner(proj xgoprojs.Proj, projectDir string) (*modfile.Runner, error) {
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
	project, err := selectTargetProject(parsed.Projects, proj, projectDir)
	if err != nil {
		return nil, err
	}
	if project == nil || project.Runner == nil {
		return nil, nil
	}
	runner := project.Runner
	if err := module.CheckImportPath(runner.Path); err != nil {
		return nil, fmt.Errorf("invalid runner path %q: %w", runner.Path, err)
	}
	return runner, nil
}

func selectTargetProject(projects []*modfile.Project, proj xgoprojs.Proj, projectDir string) (*modfile.Project, error) {
	switch len(projects) {
	case 0:
		return nil, nil
	case 1:
		return projects[0], nil
	}

	targetFilenames, err := collectTargetFilenames(proj, projectDir)
	if err != nil {
		return nil, err
	}

	gopModPath := filepath.Join(projectDir, "gop.mod")
	var matched *modfile.Project
	for _, filename := range targetFilenames {
		ext := modfile.ClassExt(filename)
		for _, project := range projects {
			if ext == project.Ext && project.IsProj(ext, filename) {
				if matched != nil && matched != project {
					return nil, fmt.Errorf("multiple projects in %s match run target", gopModPath)
				}
				matched = project
			}
		}
	}
	return matched, nil
}

func collectTargetFilenames(proj xgoprojs.Proj, projectDir string) ([]string, error) {
	switch v := proj.(type) {
	case *xgoprojs.FilesProj:
		filenames := make([]string, 0, len(v.Files))
		for _, file := range v.Files {
			filenames = append(filenames, filepath.Base(file))
		}
		return filenames, nil
	case *xgoprojs.DirProj, *xgoprojs.PkgPathProj:
		entries, err := os.ReadDir(projectDir)
		if err != nil {
			return nil, err
		}
		files := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			files = append(files, entry.Name())
		}
		return files, nil
	default:
		return nil, fmt.Errorf("unsupported project type %T", proj)
	}
}

func readProjectGopMod(projectDir string) (string, []byte, error) {
	gopModPath := filepath.Join(projectDir, "gop.mod")
	data, err := os.ReadFile(gopModPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return gopModPath, nil, nil
		}
		return "", nil, err
	}
	return gopModPath, data, nil
}
