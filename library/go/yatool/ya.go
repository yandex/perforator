package yatool

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	ya     string
	yaErr  error
	yaOnce sync.Once
)

const (
	windowsOS = "windows"
)

// FindYa returns path to ya for arcadia root at the target path
func FindYa(path string) (string, error) {
	arcadiaRoot, err := FindArcadiaRoot(path)
	if err != nil {
		return "", err
	}

	return yaForPath(arcadiaRoot)
}

// Ya returns "ya" path for current Arcadia root
func Ya() (string, error) {
	yaOnce.Do(func() {
		var arcadiaRoot string
		arcadiaRoot, yaErr = ArcadiaRoot()
		if yaErr != nil {
			return
		}

		ya, yaErr = yaForPath(arcadiaRoot)
	})

	return ya, yaErr
}

func yaForPath(path string) (string, error) {
	yaBinary := "ya"
	if runtime.GOOS == windowsOS {
		yaBinary += ".bat"
	}

	yaPath := filepath.Join(path, yaBinary)
	if _, err := os.Stat(yaPath); err != nil {
		return "", err
	}

	return yaPath, nil
}
