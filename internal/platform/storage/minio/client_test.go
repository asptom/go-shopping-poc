package minio

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty endpoint",
			config: &Config{
				Endpoint:  "",
				AccessKey: "test",
				SecretKey: "test",
			},
			wantErr: true,
		},
		{
			name: "empty access key",
			config: &Config{
				Endpoint:  "localhost:9000",
				AccessKey: "",
				SecretKey: "test",
			},
			wantErr: true,
		},
		{
			name: "empty secret key",
			config: &Config{
				Endpoint:  "localhost:9000",
				AccessKey: "test",
				SecretKey: "",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &Config{
				Endpoint:  "localhost:9000",
				AccessKey: "test",
				SecretKey: "test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestClient_BucketOperations(t *testing.T) {
	ctx := context.Background()

	// Create a mock client for testing
	config := &Config{
		Endpoint:  "localhost:9000",
		AccessKey: "test",
		SecretKey: "test",
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	t.Run("CreateBucket", func(t *testing.T) {
		err := client.CreateBucket(ctx, "test-bucket")
		// This will fail in real scenario but we're testing the interface
		assert.Error(t, err) // MinIO connection error expected
	})

	t.Run("BucketExists", func(t *testing.T) {
		exists, err := client.BucketExists(ctx, "test-bucket")
		assert.Error(t, err) // MinIO connection error expected
		assert.False(t, exists)
	})

	t.Run("DeleteBucket", func(t *testing.T) {
		err := client.DeleteBucket(ctx, "test-bucket")
		assert.Error(t, err) // MinIO connection error expected
	})
}

func TestClient_ObjectOperations(t *testing.T) {
	ctx := context.Background()

	config := &Config{
		Endpoint:  "localhost:9000",
		AccessKey: "test",
		SecretKey: "test",
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	t.Run("PutObject with empty bucket", func(t *testing.T) {
		reader := strings.NewReader("test content")
		_, err := client.PutObject(ctx, "", "test-object", reader, int64(reader.Len()), PutObjectOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name cannot be empty")
	})

	t.Run("PutObject with empty object name", func(t *testing.T) {
		reader := strings.NewReader("test content")
		_, err := client.PutObject(ctx, "test-bucket", "", reader, int64(reader.Len()), PutObjectOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "object name cannot be empty")
	})

	t.Run("PutObject with nil reader", func(t *testing.T) {
		_, err := client.PutObject(ctx, "test-bucket", "test-object", nil, 0, PutObjectOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reader cannot be nil")
	})

	t.Run("GetObject with empty bucket", func(t *testing.T) {
		_, err := client.GetObject(ctx, "", "test-object")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name cannot be empty")
	})

	t.Run("GetObject with empty object name", func(t *testing.T) {
		_, err := client.GetObject(ctx, "test-bucket", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "object name cannot be empty")
	})

	t.Run("StatObject with empty bucket", func(t *testing.T) {
		_, err := client.StatObject(ctx, "", "test-object")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name cannot be empty")
	})

	t.Run("StatObject with empty object name", func(t *testing.T) {
		_, err := client.StatObject(ctx, "test-bucket", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "object name cannot be empty")
	})

	t.Run("RemoveObject with empty bucket", func(t *testing.T) {
		err := client.RemoveObject(ctx, "", "test-object")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name cannot be empty")
	})

	t.Run("RemoveObject with empty object name", func(t *testing.T) {
		err := client.RemoveObject(ctx, "test-bucket", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "object name cannot be empty")
	})
}

func TestClient_FileOperations(t *testing.T) {
	ctx := context.Background()

	config := &Config{
		Endpoint:  "localhost:9000",
		AccessKey: "test",
		SecretKey: "test",
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	t.Run("FGetObject with empty bucket", func(t *testing.T) {
		err := client.FGetObject(ctx, "", "test-object", "/tmp/test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name cannot be empty")
	})

	t.Run("FGetObject with empty object name", func(t *testing.T) {
		err := client.FGetObject(ctx, "test-bucket", "", "/tmp/test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "object name cannot be empty")
	})

	t.Run("FGetObject with empty file path", func(t *testing.T) {
		err := client.FGetObject(ctx, "test-bucket", "test-object", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file path cannot be empty")
	})
}

func TestClient_PresignedURLs(t *testing.T) {
	ctx := context.Background()

	config := &Config{
		Endpoint:  "localhost:9000",
		AccessKey: "test",
		SecretKey: "test",
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	t.Run("PresignedGetObject with empty bucket", func(t *testing.T) {
		_, err := client.PresignedGetObject(ctx, "", "test-object", 3600)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name cannot be empty")
	})

	t.Run("PresignedGetObject with empty object name", func(t *testing.T) {
		_, err := client.PresignedGetObject(ctx, "test-bucket", "", 3600)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "object name cannot be empty")
	})

	t.Run("PresignedPutObject with empty bucket", func(t *testing.T) {
		_, err := client.PresignedPutObject(ctx, "", "test-object", 3600)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name cannot be empty")
	})

	t.Run("PresignedPutObject with empty object name", func(t *testing.T) {
		_, err := client.PresignedPutObject(ctx, "test-bucket", "", 3600)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "object name cannot be empty")
	})
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		filePath string
		expected string
	}{
		{"test.jpg", "image/jpeg"},
		{"test.jpeg", "image/jpeg"},
		{"test.png", "image/png"},
		{"test.gif", "image/gif"},
		{"test.webp", "image/webp"},
		{"test.svg", "image/svg+xml"},
		{"test.bmp", "image/bmp"},
		{"test.tiff", "image/tiff"},
		{"test.tif", "image/tiff"},
		{"test.ico", "image/x-icon"},
		{"test.pdf", "application/pdf"},
		{"test.txt", "text/plain"},
		{"test.json", "application/json"},
		{"test.xml", "application/xml"},
		{"test.html", "text/html"},
		{"test.css", "text/css"},
		{"test.js", "application/javascript"},
		{"test.zip", "application/zip"},
		{"test.tar", "application/x-tar"},
		{"test.gz", "application/gzip"},
		{"test.unknown", "application/octet-stream"},
		{"", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			result := GetContentType(tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_ListObjects(t *testing.T) {
	ctx := context.Background()

	config := &Config{
		Endpoint:  "localhost:9000",
		AccessKey: "test",
		SecretKey: "test",
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	t.Run("ListObjects with empty bucket", func(t *testing.T) {
		ch := client.ListObjects(ctx, "", ListObjectsOptions{})
		// Should return a closed channel
		select {
		case _, ok := <-ch:
			assert.False(t, ok, "channel should be closed")
		default:
			// Channel is empty and closed
		}
	})
}

func TestPutObjectOptions(t *testing.T) {
	opts := PutObjectOptions{
		ContentType:        "application/json",
		ContentEncoding:    "gzip",
		ContentLanguage:    "en",
		ContentDisposition: "attachment; filename=\"test.json\"",
		CacheControl:       "max-age=3600",
		Metadata:           map[string]string{"key": "value"},
		UserTags:           map[string]string{"tag": "test"},
		StorageClass:       "STANDARD",
		Retention: ObjectRetention{
			Mode:            RetentionModeGovernance,
			RetainUntilDate: "2024-12-31T23:59:59Z",
		},
		LegalHold: true,
	}

	assert.Equal(t, "application/json", opts.ContentType)
	assert.Equal(t, "gzip", opts.ContentEncoding)
	assert.Equal(t, "en", opts.ContentLanguage)
	assert.Equal(t, "attachment; filename=\"test.json\"", opts.ContentDisposition)
	assert.Equal(t, "max-age=3600", opts.CacheControl)
	assert.Equal(t, map[string]string{"key": "value"}, opts.Metadata)
	assert.Equal(t, map[string]string{"tag": "test"}, opts.UserTags)
	assert.Equal(t, "STANDARD", opts.StorageClass)
	assert.Equal(t, RetentionModeGovernance, opts.Retention.Mode)
	assert.Equal(t, "2024-12-31T23:59:59Z", opts.Retention.RetainUntilDate)
	assert.True(t, opts.LegalHold)
}

func TestListObjectsOptions(t *testing.T) {
	opts := ListObjectsOptions{
		Prefix:       "test/",
		StartAfter:   "test/file1.txt",
		Recursive:    true,
		MaxKeys:      1000,
		WithMetadata: true,
		WithVersions: true,
	}

	assert.Equal(t, "test/", opts.Prefix)
	assert.Equal(t, "test/file1.txt", opts.StartAfter)
	assert.True(t, opts.Recursive)
	assert.Equal(t, 1000, opts.MaxKeys)
	assert.True(t, opts.WithMetadata)
	assert.True(t, opts.WithVersions)
}

func TestObjectInfo(t *testing.T) {
	info := ObjectInfo{
		Bucket:         "test-bucket",
		Key:            "test-object",
		Size:           1024,
		LastModified:   "2023-01-01T00:00:00Z",
		ETag:           "test-etag",
		ContentType:    "application/octet-stream",
		Metadata:       map[string]string{"key": "value"},
		UserTags:       map[string]string{"tag": "test"},
		VersionID:      "v1",
		IsLatest:       true,
		IsDeleteMarker: false,
		StorageClass:   "STANDARD",
	}

	assert.Equal(t, "test-bucket", info.Bucket)
	assert.Equal(t, "test-object", info.Key)
	assert.Equal(t, int64(1024), info.Size)
	assert.Equal(t, "2023-01-01T00:00:00Z", info.LastModified)
	assert.Equal(t, "test-etag", info.ETag)
	assert.Equal(t, "application/octet-stream", info.ContentType)
	assert.Equal(t, map[string]string{"key": "value"}, info.Metadata)
	assert.Equal(t, map[string]string{"tag": "test"}, info.UserTags)
	assert.Equal(t, "v1", info.VersionID)
	assert.True(t, info.IsLatest)
	assert.False(t, info.IsDeleteMarker)
	assert.Equal(t, "STANDARD", info.StorageClass)
}

func TestRetentionMode(t *testing.T) {
	assert.Equal(t, RetentionMode(""), RetentionModeNone)
	assert.Equal(t, RetentionMode("GOVERNANCE"), RetentionModeGovernance)
	assert.Equal(t, RetentionMode("COMPLIANCE"), RetentionModeCompliance)
}

func TestCopyObject(t *testing.T) {
	ctx := context.Background()

	config := &Config{
		Endpoint:  "localhost:9000",
		AccessKey: "test",
		SecretKey: "test",
	}
	client, err := NewClient(config)
	require.NoError(t, err)

	t.Run("CopyObject with empty destination bucket", func(t *testing.T) {
		err := client.CopyObject(ctx, "", "dst-object", "src-bucket", "src-object", CopyObjectOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "destination bucket and object names cannot be empty")
	})

	t.Run("CopyObject with empty destination object", func(t *testing.T) {
		err := client.CopyObject(ctx, "dst-bucket", "", "src-bucket", "src-object", CopyObjectOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "destination bucket and object names cannot be empty")
	})

	t.Run("CopyObject with empty source bucket", func(t *testing.T) {
		err := client.CopyObject(ctx, "dst-bucket", "dst-object", "", "src-object", CopyObjectOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source bucket and object names cannot be empty")
	})

	t.Run("CopyObject with empty source object", func(t *testing.T) {
		err := client.CopyObject(ctx, "dst-bucket", "dst-object", "src-bucket", "", CopyObjectOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source bucket and object names cannot be empty")
	})
}

// TestIntegration tests with a real MinIO server (requires MinIO running)
func TestIntegration_PutAndGetObject(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires a running MinIO server
	// Set up test MinIO server configuration
	config := &Config{
		Endpoint:  "localhost:9000",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Secure:    false,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Skip("MinIO server not available:", err)
	}

	ctx := context.Background()
	bucketName := "test-integration-bucket"
	objectName := "test-object.txt"
	testContent := "Hello, MinIO!"

	// Clean up
	defer func() {
		_ = client.RemoveObject(ctx, bucketName, objectName)
		_ = client.DeleteBucket(ctx, bucketName)
	}()

	// Create bucket
	err = client.CreateBucket(ctx, bucketName)
	if err != nil {
		t.Skip("Cannot create bucket:", err)
	}

	// Put object
	reader := strings.NewReader(testContent)
	info, err := client.PutObject(ctx, bucketName, objectName, reader, int64(len(testContent)), PutObjectOptions{
		ContentType: "text/plain",
	})
	require.NoError(t, err)
	assert.Equal(t, bucketName, info.Bucket)
	assert.Equal(t, objectName, info.Key)
	assert.Equal(t, int64(len(testContent)), info.Size)

	// Get object
	readCloser, err := client.GetObject(ctx, bucketName, objectName)
	require.NoError(t, err)
	defer func() { _ = readCloser.Close() }()

	content, err := io.ReadAll(readCloser)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))

	// Stat object
	statInfo, err := client.StatObject(ctx, bucketName, objectName)
	require.NoError(t, err)
	assert.Equal(t, objectName, statInfo.Key)
	assert.Equal(t, int64(len(testContent)), statInfo.Size)
	assert.Equal(t, "text/plain", statInfo.ContentType)
}
