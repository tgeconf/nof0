package dotenv

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

// loadDotenv attempts to load environment variables from a .env file.
// Priority:
// 1) ENV_FILE if set (single path)
// 2) .env relative to this source file, then one and two directories up
// 3) .env in current working directory (as a final fallback)
// Skips when NO_DOTENV=1.
func loadDotenv() {
	if os.Getenv("NO_DOTENV") == "1" {
		return
	}

	// Default behavior: do NOT override existing OS/CI variables.
	// Set DOTENV_OVERLOAD=1 to flip behavior and force .env to win.
	overload := os.Getenv("DOTENV_OVERLOAD") == "1"
	load := func(paths ...string) {
		if overload {
			_ = godotenv.Overload(paths...)
		} else {
			_ = godotenv.Load(paths...)
		}
	}

	if envFile := os.Getenv("ENV_FILE"); envFile != "" {
		load(envFile)
		return
	}

	if _, file, _, ok := runtime.Caller(0); ok {
		// Walk up directories to locate repo root (presence of go.mod or .git)
		dir := filepath.Dir(file)
		for i := 0; i < 8; i++ { // up to 8 levels to be safe
			// Try .env at this level
			load(filepath.Join(dir, ".env"))
			// If go.mod or .git exists here, stop after attempting .env
			if exists(filepath.Join(dir, "go.mod")) || exists(filepath.Join(dir, ".git")) {
				return
			}
			parent := filepath.Dir(dir)
			if parent == dir { // reached root
				break
			}
			dir = parent
		}
		return
	}

	load(".env")
}

func exists(p string) bool {
	if _, err := os.Stat(p); err == nil {
		return true
	}
	return false
}
