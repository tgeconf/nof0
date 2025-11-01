package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// ProjectRoot attempts to locate the repository root by walking upward from
// the caller location until it finds a directory containing go.mod or .git.
// Returns the working directory on failure.
func ProjectRoot() (string, error) {
	if _, file, _, ok := runtime.Caller(0); ok {
		dir := filepath.Dir(file)
		for i := 0; i < 8; i++ {
			if exists(filepath.Join(dir, "go.mod")) || exists(filepath.Join(dir, ".git")) {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	// Fallback to current working directory.
	wd, err := os.Getwd()
	if err != nil {
		return ".", err
	}
	return wd, nil
}

// MustProjectRoot is like ProjectRoot but panics on failure.
func MustProjectRoot() string {
	root, _ := ProjectRoot()
	return root
}

// ProjectPath joins the project root with the provided relative path.
func ProjectPath(rel string) (string, error) {
	root, err := ProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, rel), nil
}

// MustProjectPath is like ProjectPath but panics on failure.
func MustProjectPath(rel string) string {
	p, _ := ProjectPath(rel)
	return p
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }
