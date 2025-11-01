package confkit_test

import (
	"os"
	"path/filepath"
	"testing"

	"nof0-api/pkg/confkit"
)

func TestResolvePath(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		file     string
		expected string
		setupEnv map[string]string
	}{
		{
			name:     "absolute path",
			base:     "/base/dir",
			file:     "/absolute/path/file.yaml",
			expected: "/absolute/path/file.yaml",
		},
		{
			name:     "relative path",
			base:     "/base/dir",
			file:     "config/file.yaml",
			expected: "/base/dir/config/file.yaml",
		},
		{
			name:     "path with env var",
			base:     "/base/dir",
			file:     "$HOME/config/file.yaml",
			expected: os.Getenv("HOME") + "/config/file.yaml",
		},
		{
			name:     "relative path with env var",
			base:     "/base/dir",
			file:     "${TEST_VAR}/file.yaml",
			expected: "testvalue/file.yaml",
			setupEnv: map[string]string{"TEST_VAR": "testvalue"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variables
			for k, v := range tt.setupEnv {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result := confkit.ResolvePath(tt.base, tt.file)

			// For relative paths with env vars, we need to handle the base concatenation
			if tt.setupEnv != nil && !filepath.IsAbs(tt.file) {
				expected := filepath.Join(tt.base, os.ExpandEnv(tt.file))
				if result != expected {
					t.Errorf("ResolvePath() = %v, want %v", result, expected)
				}
			} else if result != tt.expected {
				t.Errorf("ResolvePath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBaseDir(t *testing.T) {
	tests := []struct {
		name     string
		mainPath string
		expected string
	}{
		{
			name:     "simple path",
			mainPath: "/etc/config/app.yaml",
			expected: "/etc/config",
		},
		{
			name:     "root path",
			mainPath: "/app.yaml",
			expected: "/",
		},
		{
			name:     "relative path",
			mainPath: "config/app.yaml",
			expected: "config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := confkit.BaseDir(tt.mainPath)
			if result != tt.expected {
				t.Errorf("BaseDir() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSection_Hydrate(t *testing.T) {
	t.Run("empty file", func(t *testing.T) {
		section := &confkit.Section[string]{}
		err := section.Hydrate("/base", func(path string) (*string, error) {
			t.Error("loader should not be called for empty file")
			return nil, nil
		})
		if err != nil {
			t.Errorf("Hydrate() with empty file should not error, got: %v", err)
		}
		if section.Value != nil {
			t.Error("Value should remain nil for empty file")
		}
	})

	t.Run("successful hydration", func(t *testing.T) {
		section := &confkit.Section[string]{File: "config.yaml"}
		expected := "test value"

		err := section.Hydrate("/base", func(path string) (*string, error) {
			if path != "/base/config.yaml" {
				t.Errorf("loader received path %v, want /base/config.yaml", path)
			}
			return &expected, nil
		})

		if err != nil {
			t.Errorf("Hydrate() error = %v, want nil", err)
		}
		if section.Value == nil || *section.Value != expected {
			t.Errorf("Value = %v, want %v", section.Value, expected)
		}
		if section.File != "/base/config.yaml" {
			t.Errorf("File = %v, want /base/config.yaml", section.File)
		}
	})
}
