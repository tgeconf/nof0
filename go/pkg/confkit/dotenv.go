package confkit

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/joho/godotenv"
)

var dotenvOnce sync.Once

// LoadDotenvOnce loads environment variables from a .env file using the same
// search semantics as the legacy bootstrap package. The first successful call
// wins; subsequent calls are no-ops. Existing environment variables are left
// untouched unless DOTENV_OVERLOAD=1 is set.
func LoadDotenvOnce() {
	dotenvOnce.Do(func() {
		loadDotenv()
	})
}

func loadDotenv() {
	if os.Getenv("NO_DOTENV") == "1" {
		return
	}

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
		dir := filepath.Dir(file)
		for i := 0; i < 8; i++ {
			load(filepath.Join(dir, ".env"))
			if fileExists(filepath.Join(dir, "go.mod")) || fileExists(filepath.Join(dir, ".git")) {
				return
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
		return
	}

	load(".env")
}
