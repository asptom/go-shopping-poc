/*
Package config_test provides tests for environment file resolution logic.

This file specifically tests the ResolveEnvFile function which determines
which .env file to load based on the APP_ENV environment variable.
*/
package config

import (
	"strings"
	"testing"
)

// ===== ENVIRONMENT FILE RESOLUTION =====

// TestResolveEnvFile_Production tests production environment file resolution
func TestResolveEnvFile_Production(t *testing.T) {
	setupTestEnv(t, map[string]string{"APP_ENV": "production"})

	result := ResolveEnvFile()
	// Should be absolute path ending with .env.production
	if !strings.HasSuffix(result, ".env.production") {
		t.Errorf("ResolveEnvFile() = %s, expected to end with .env.production", result)
	}
}

// TestResolveEnvFile_Staging tests staging environment file resolution
func TestResolveEnvFile_Staging(t *testing.T) {
	setupTestEnv(t, map[string]string{"APP_ENV": "staging"})

	result := ResolveEnvFile()
	// Should be absolute path ending with .env.staging
	if !strings.HasSuffix(result, ".env.staging") {
		t.Errorf("ResolveEnvFile() = %s, expected to end with .env.staging", result)
	}
}

// TestResolveEnvFile_Test tests test environment file resolution
func TestResolveEnvFile_Test(t *testing.T) {
	setupTestEnv(t, map[string]string{"APP_ENV": "test"})

	result := ResolveEnvFile()
	// Should be absolute path ending with .env.test
	if !strings.HasSuffix(result, ".env.test") {
		t.Errorf("ResolveEnvFile() = %s, expected to end with .env.test", result)
	}
}

// TestResolveEnvFile_Default tests default environment file resolution
func TestResolveEnvFile_Default(t *testing.T) {
	// Ensure APP_ENV is not set
	setupTestEnv(t, map[string]string{"APP_ENV": ""})

	result := ResolveEnvFile()
	// Should be absolute path ending with .env.local
	if !strings.HasSuffix(result, ".env.local") {
		t.Errorf("ResolveEnvFile() = %s, expected to end with .env.local", result)
	}
}

// TestResolveEnvFile_UnknownValue tests unknown APP_ENV value defaults to local
func TestResolveEnvFile_UnknownValue(t *testing.T) {
	setupTestEnv(t, map[string]string{"APP_ENV": "development"})

	result := ResolveEnvFile()
	// Should be absolute path ending with .env.local
	if !strings.HasSuffix(result, ".env.local") {
		t.Errorf("ResolveEnvFile() = %s, expected to end with .env.local", result)
	}
}
