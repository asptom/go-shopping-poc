package config

import (
	"os"
	"path/filepath"
)

// findProjectRoot finds the project root by looking for go.mod
func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}
		dir = parent
	}

	return ""
}

// ResolveEnvFile returns the correct .env file path based on APP_ENV.
func ResolveEnvFile() string {
	env := os.Getenv("APP_ENV")

	var filename string
	switch env {
	case "production":
		filename = ".env.production"
	case "staging":
		filename = ".env.staging"
	case "test":
		filename = ".env.test"
	default:
		filename = ".env.local"
	}

	// Find project root and return absolute path
	if root := findProjectRoot(); root != "" {
		return filepath.Join(root, filename)
	}

	// Fallback to relative path if project root not found
	return filename
}
