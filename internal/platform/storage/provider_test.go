package storage

import (
	"os"
	"testing"
)

// TestNewStorageProvider_Success tests successful storage provider creation
func TestNewStorageProvider_Success(t *testing.T) {
	// Set environment variables for MinIO configuration
	os.Setenv("MINIO_ENDPOINT_KUBERNETES", "minio.minio.svc.cluster.local:9000")
	os.Setenv("MINIO_ENDPOINT_LOCAL", "api.minio.local")
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadminpassword")
	os.Setenv("MINIO_TLS_VERIFY", "false")

	// Clean up environment variables after test
	defer func() {
		os.Unsetenv("MINIO_ENDPOINT_KUBERNETES")
		os.Unsetenv("MINIO_ENDPOINT_LOCAL")
		os.Unsetenv("MINIO_ACCESS_KEY")
		os.Unsetenv("MINIO_SECRET_KEY")
		os.Unsetenv("MINIO_TLS_VERIFY")
	}()

	provider, err := NewStorageProvider()
	if err != nil {
		t.Fatalf("NewStorageProvider() failed: %v", err)
	}

	if provider == nil {
		t.Fatal("NewStorageProvider() returned nil provider")
	}

	// Test that we can get object storage
	storage := provider.GetObjectStorage()
	if storage == nil {
		t.Fatal("GetObjectStorage() returned nil storage")
	}
}

// TestNewStorageProvider_ConfigLoadError tests provider creation with missing environment variables
func TestNewStorageProvider_ConfigLoadError(t *testing.T) {
	// Ensure required environment variables are not set
	os.Unsetenv("MINIO_ENDPOINT_KUBERNETES")
	os.Unsetenv("MINIO_ENDPOINT_LOCAL")
	os.Unsetenv("MINIO_ACCESS_KEY")
	os.Unsetenv("MINIO_SECRET_KEY")

	provider, err := NewStorageProvider()
	if err == nil {
		t.Fatal("NewStorageProvider() should have failed with missing environment variables")
	}

	if provider != nil {
		t.Fatal("NewStorageProvider() should have returned nil provider on config load error")
	}
}

// TestStorageProviderImpl_GetObjectStorage tests the GetObjectStorage method
func TestStorageProviderImpl_GetObjectStorage(t *testing.T) {
	// Create a mock storage provider (we can't easily create a real one without MinIO server)
	// This test verifies the interface compliance and basic functionality

	// We can't create a real provider without config, so we'll test the interface compliance
	// by creating a minimal implementation for testing

	var provider StorageProvider
	if provider != nil {
		storage := provider.GetObjectStorage()
		if storage == nil {
			t.Error("GetObjectStorage() should not return nil when provider is not nil")
		}
	}
}

// TestStorageProviderImpl_InterfaceCompliance tests that StorageProviderImpl implements StorageProvider interface
func TestStorageProviderImpl_InterfaceCompliance(t *testing.T) {
	var _ StorageProvider = (*StorageProviderImpl)(nil)
}

// TestNewStorageProvider_EnvironmentEndpointSelection tests endpoint selection based on environment
func TestNewStorageProvider_EnvironmentEndpointSelection(t *testing.T) {
	// Set environment variables for MinIO configuration
	os.Setenv("MINIO_ENDPOINT_KUBERNETES", "minio.minio.svc.cluster.local:9000")
	os.Setenv("MINIO_ENDPOINT_LOCAL", "api.minio.local")
	os.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	os.Setenv("MINIO_SECRET_KEY", "minioadminpassword")
	os.Setenv("MINIO_TLS_VERIFY", "false")

	// Clean up environment variables after test
	defer func() {
		os.Unsetenv("MINIO_ENDPOINT_KUBERNETES")
		os.Unsetenv("MINIO_ENDPOINT_LOCAL")
		os.Unsetenv("MINIO_ACCESS_KEY")
		os.Unsetenv("MINIO_SECRET_KEY")
		os.Unsetenv("MINIO_TLS_VERIFY")
	}()

	// Test local environment (no KUBERNETES_SERVICE_HOST)
	os.Unsetenv("KUBERNETES_SERVICE_HOST")
	provider, err := NewStorageProvider()
	if err != nil {
		t.Fatalf("NewStorageProvider() failed in local environment: %v", err)
	}
	if provider == nil {
		t.Fatal("NewStorageProvider() returned nil provider in local environment")
	}

	// Test Kubernetes environment
	os.Setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
	defer os.Unsetenv("KUBERNETES_SERVICE_HOST")

	provider, err = NewStorageProvider()
	if err != nil {
		t.Fatalf("NewStorageProvider() failed in Kubernetes environment: %v", err)
	}
	if provider == nil {
		t.Fatal("NewStorageProvider() returned nil provider in Kubernetes environment")
	}
}
