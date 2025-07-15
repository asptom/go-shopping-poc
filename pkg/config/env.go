package config

import (
	"os"
)

// ResolveEnvFile returns the correct .env file path based on APP_ENV.
func ResolveEnvFile() string {

	env := os.Getenv("APP_ENV")

	switch env {
	case "production":
		return ".env.production"
	case "staging":
		return ".env.staging"
	case "test":
		return ".env.test"
	default:
		return ".env.development"
	}
}
