package confkit

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ProjectRoot attempts to locate the repository root by walking upwards from
// this source file until it finds a directory containing go.mod or .git.
// Falls back to the current working directory on failure.
func ProjectRoot() (string, error) {
	if _, file, _, ok := runtime.Caller(0); ok {
		dir := filepath.Dir(file)
		for i := 0; i < 8; i++ {
			if fileExists(filepath.Join(dir, "go.mod")) || fileExists(filepath.Join(dir, ".git")) {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return ".", fmt.Errorf("getwd: %w", err)
	}
	return wd, nil
}

// MustProjectRoot returns the repository root path or panics on failure.
func MustProjectRoot() string {
	root, err := ProjectRoot()
	if err != nil {
		panic(err)
	}
	return root
}

// ProjectPath joins the repository root with the provided relative path.
func ProjectPath(rel string) (string, error) {
	root, err := ProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, rel), nil
}

// MustProjectPath returns ProjectPath(rel) and panics on failure.
func MustProjectPath(rel string) string {
	p, err := ProjectPath(rel)
	if err != nil {
		panic(err)
	}
	return p
}
