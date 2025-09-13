package yatool

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
)

var (
	arcadiaRoot     string
	arcadiaRootErr  error
	arcadiaRootOnce sync.Once

	goModDir     string
	goModDirErr  error
	goModDirOnce sync.Once
)

// FindArcadiaRoot searches Arcadia root for the target path
//
// Implementation reference https://github.com/yandex/perforator/arc/trunk/arcadia/library/python/find_root
func FindArcadiaRoot(arcPath string) (string, error) {
	isRoot := func(arcPath string) bool {
		if _, err := os.Stat(filepath.Join(arcPath, ".arcadia.root")); err == nil {
			return true
		}

		if _, err := os.Stat(filepath.Join(arcPath, "devtools", "ya", "ya.conf.json")); err == nil {
			return true
		}

		return false
	}

	arcPath, err := filepath.Abs(arcPath)
	if err != nil {
		return "", err
	}

	arcPath, err = filepath.EvalSymlinks(arcPath)
	if err != nil {
		return "", err
	}

	current := filepath.Clean(arcPath)
	for {
		if isRoot(current) {
			return current, nil
		}

		next := filepath.Dir(current)
		if next == current {
			return "", errors.New("can't find arcadia root")
		}

		current = next
	}
}

// ArcadiaRoot returns the current Arcadia root
func ArcadiaRoot() (string, error) {
	arcadiaRootOnce.Do(func() {
		arcadiaRoot, arcadiaRootErr = FindArcadiaRoot(".")
	})

	return arcadiaRoot, arcadiaRootErr
}

func findGoModDir(arcPath string) (string, error) {
	isRoot := func(arcPath string) (bool, string) {
		if _, err := os.Stat(filepath.Join(arcPath, ".arcadia.root")); err == nil {
			return true, arcPath
		}

		// Checking for cloudia/cloud/cloud-go monorepo
		if _, err := os.Stat(filepath.Join(arcPath, ".cloudia.root")); err == nil {
			return true, filepath.Join(arcPath, "cloud", "cloud-go")
		}

		return false, ""
	}

	arcPath, err := filepath.Abs(arcPath)
	if err != nil {
		return "", err
	}

	current := filepath.Clean(arcPath)
	for {
		root, goModPath := isRoot(current)
		if root {
			return goModPath, nil
		}

		next := filepath.Dir(current)
		if next == current {
			return "", errors.New("can't find arcadia root")
		}

		current = next
	}
}

func FindRepositoryGoModDir() (string, error) {
	goModDirOnce.Do(func() {
		goModDir, goModDirErr = findGoModDir(".")
	})

	return goModDir, goModDirErr
}
