package config

import (
	"os"
	"path/filepath"
)

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

	// Return relative path to config/ folder
	return filepath.Join("config", filename)
}
