package minio

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client wraps the MinIO client with additional functionality
type Client struct {
	client *minio.Client
	config *Config
}

// Config holds MinIO client configuration
type Config struct {
	Endpoint     string
	AccessKey    string
	SecretKey    string
	SessionToken string
	Secure       bool
	Region       string
	MaxRetries   int
}

// NewClient creates a new MinIO client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}

	if config.AccessKey == "" {
		return nil, fmt.Errorf("access key is required")
	}

	if config.SecretKey == "" {
		return nil, fmt.Errorf("secret key is required")
	}

	// Set defaults
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	// Create MinIO client
	minioClient, err := minio.New(config.Endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(config.AccessKey, config.SecretKey, config.SessionToken),
		Secure:    config.Secure,
		Region:    config.Region,
		Transport: nil, // Use default transport
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	client := &Client{
		client: minioClient,
		config: config,
	}

	return client, nil
}

// CreateBucket creates a new bucket
func (c *Client) CreateBucket(ctx context.Context, bucketName string) error {
	if bucketName == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}

	err := c.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
		Region: c.config.Region,
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
	}

	return nil
}

// BucketExists checks if a bucket exists
func (c *Client) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	if bucketName == "" {
		return false, fmt.Errorf("bucket name cannot be empty")
	}

	exists, err := c.client.BucketExists(ctx, bucketName)
	if err != nil {
		return false, fmt.Errorf("failed to check bucket existence %s: %w", bucketName, err)
	}

	return exists, nil
}

// DeleteBucket deletes a bucket
func (c *Client) DeleteBucket(ctx context.Context, bucketName string) error {
	if bucketName == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}

	err := c.client.RemoveBucket(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to delete bucket %s: %w", bucketName, err)
	}

	return nil
}

// PutObject uploads an object to the bucket
func (c *Client) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts PutObjectOptions) (ObjectInfo, error) {
	if bucketName == "" {
		return ObjectInfo{}, fmt.Errorf("bucket name cannot be empty")
	}
	if objectName == "" {
		return ObjectInfo{}, fmt.Errorf("object name cannot be empty")
	}
	if reader == nil {
		return ObjectInfo{}, fmt.Errorf("reader cannot be nil")
	}

	minioOpts := c.convertPutOptions(opts)

	info, err := c.client.PutObject(ctx, bucketName, objectName, reader, objectSize, minioOpts)
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("failed to put object %s/%s: %w", bucketName, objectName, err)
	}

	return c.convertUploadInfo(bucketName, objectName, info), nil
}

// GetObject retrieves an object from the bucket
func (c *Client) GetObject(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if objectName == "" {
		return nil, fmt.Errorf("object name cannot be empty")
	}

	object, err := c.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s/%s: %w", bucketName, objectName, err)
	}

	return object, nil
}

// StatObject gets object information
func (c *Client) StatObject(ctx context.Context, bucketName, objectName string) (ObjectInfo, error) {
	if bucketName == "" {
		return ObjectInfo{}, fmt.Errorf("bucket name cannot be empty")
	}
	if objectName == "" {
		return ObjectInfo{}, fmt.Errorf("object name cannot be empty")
	}

	info, err := c.client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("failed to stat object %s/%s: %w", bucketName, objectName, err)
	}

	return c.convertObjectInfo(info), nil
}

// RemoveObject removes an object from the bucket
func (c *Client) RemoveObject(ctx context.Context, bucketName, objectName string) error {
	if bucketName == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}
	if objectName == "" {
		return fmt.Errorf("object name cannot be empty")
	}

	err := c.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove object %s/%s: %w", bucketName, objectName, err)
	}

	return nil
}

// CopyObject copies an object within or between buckets
func (c *Client) CopyObject(ctx context.Context, dstBucket, dstObject, srcBucket, srcObject string, opts CopyObjectOptions) error {
	if dstBucket == "" || dstObject == "" {
		return fmt.Errorf("destination bucket and object names cannot be empty")
	}
	if srcBucket == "" || srcObject == "" {
		return fmt.Errorf("source bucket and object names cannot be empty")
	}

	srcOpts := minio.CopySrcOptions{
		Bucket: srcBucket,
		Object: srcObject,
	}

	dstOpts := minio.CopyDestOptions{
		Bucket:          dstBucket,
		Object:          dstObject,
		ReplaceMetadata: opts.ReplaceMetadata,
		UserMetadata:    opts.Metadata,
		UserTags:        opts.UserTags,
	}

	_, err := c.client.CopyObject(ctx, dstOpts, srcOpts)
	if err != nil {
		return fmt.Errorf("failed to copy object from %s/%s to %s/%s: %w", srcBucket, srcObject, dstBucket, dstObject, err)
	}

	return nil
}

// FPutObject uploads a file to the bucket
func (c *Client) FPutObject(ctx context.Context, bucketName, objectName, filePath string, opts PutObjectOptions) (ObjectInfo, error) {
	if bucketName == "" {
		return ObjectInfo{}, fmt.Errorf("bucket name cannot be empty")
	}
	if objectName == "" {
		return ObjectInfo{}, fmt.Errorf("object name cannot be empty")
	}
	if filePath == "" {
		return ObjectInfo{}, fmt.Errorf("file path cannot be empty")
	}

	minioOpts := c.convertPutOptions(opts)

	info, err := c.client.FPutObject(ctx, bucketName, objectName, filePath, minioOpts)
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("failed to upload file %s to %s/%s: %w", filePath, bucketName, objectName, err)
	}

	return c.convertUploadInfo(bucketName, objectName, info), nil
}

// FGetObject downloads an object to a file
func (c *Client) FGetObject(ctx context.Context, bucketName, objectName, filePath string) error {
	if bucketName == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}
	if objectName == "" {
		return fmt.Errorf("object name cannot be empty")
	}
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	err := c.client.FGetObject(ctx, bucketName, objectName, filePath, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to download object %s/%s to %s: %w", bucketName, objectName, filePath, err)
	}

	return nil
}

// ListObjects lists objects in a bucket
func (c *Client) ListObjects(ctx context.Context, bucketName string, opts ListObjectsOptions) <-chan ObjectInfo {
	if bucketName == "" {
		ch := make(chan ObjectInfo)
		close(ch) // Close immediately for empty bucket
		return ch
	}

	minioOpts := c.convertListOptions(opts)

	objectCh := c.client.ListObjects(ctx, bucketName, minioOpts)
	infoCh := make(chan ObjectInfo, 100) // Buffered channel for performance

	go func() {
		defer close(infoCh)
		for object := range objectCh {
			if object.Err != nil {
				// Send error as empty object info
				infoCh <- ObjectInfo{}
				return
			}
			infoCh <- c.convertListObjectInfo(object)
		}
	}()

	return infoCh
}

// PresignedGetObject generates a presigned GET URL for an object
func (c *Client) PresignedGetObject(ctx context.Context, bucketName, objectName string, expirySeconds int64) (string, error) {
	if bucketName == "" {
		return "", fmt.Errorf("bucket name cannot be empty")
	}
	if objectName == "" {
		return "", fmt.Errorf("object name cannot be empty")
	}
	if objectName == "" {
		return "", fmt.Errorf("object name cannot be empty")
	}
	if expirySeconds <= 0 {
		expirySeconds = 3600 // Default 1 hour
	}

	reqParams := make(url.Values)
	presignedURL, err := c.client.PresignedGetObject(ctx, bucketName, objectName, time.Duration(expirySeconds)*time.Second, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned GET URL for %s/%s: %w", bucketName, objectName, err)
	}

	return presignedURL.String(), nil
}

// PresignedPutObject generates a presigned PUT URL for an object
func (c *Client) PresignedPutObject(ctx context.Context, bucketName, objectName string, expirySeconds int64) (string, error) {
	if bucketName == "" {
		return "", fmt.Errorf("bucket name cannot be empty")
	}
	if objectName == "" {
		return "", fmt.Errorf("object name cannot be empty")
	}
	if expirySeconds <= 0 {
		expirySeconds = 3600 // Default 1 hour
	}

	presignedURL, err := c.client.PresignedPutObject(ctx, bucketName, objectName, time.Duration(expirySeconds)*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned PUT URL for %s/%s: %w", bucketName, objectName, err)
	}

	return presignedURL.String(), nil
}

// convertPutOptions converts our PutObjectOptions to minio.PutObjectOptions
func (c *Client) convertPutOptions(opts PutObjectOptions) minio.PutObjectOptions {
	minioOpts := minio.PutObjectOptions{
		ContentType:        opts.ContentType,
		ContentEncoding:    opts.ContentEncoding,
		ContentLanguage:    opts.ContentLanguage,
		ContentDisposition: opts.ContentDisposition,
		CacheControl:       opts.CacheControl,
		UserMetadata:       opts.Metadata,
		UserTags:           opts.UserTags,
		StorageClass:       opts.StorageClass,
	}

	if opts.Retention.Mode != "" {
		minioOpts.Mode = minio.RetentionMode(opts.Retention.Mode)
		if opts.Retention.RetainUntilDate != "" {
			if retainUntil, err := time.Parse(time.RFC3339, opts.Retention.RetainUntilDate); err == nil {
				minioOpts.RetainUntilDate = retainUntil
			}
		}
	}

	if opts.LegalHold {
		minioOpts.LegalHold = minio.LegalHoldEnabled
	}

	return minioOpts
}

// convertListOptions converts our ListObjectsOptions to minio.ListObjectsOptions
func (c *Client) convertListOptions(opts ListObjectsOptions) minio.ListObjectsOptions {
	minioOpts := minio.ListObjectsOptions{
		Prefix:       opts.Prefix,
		StartAfter:   opts.StartAfter,
		Recursive:    opts.Recursive,
		MaxKeys:      opts.MaxKeys,
		WithMetadata: opts.WithMetadata,
		WithVersions: opts.WithVersions,
	}

	return minioOpts
}

// convertUploadInfo converts minio.UploadInfo to our ObjectInfo
func (c *Client) convertUploadInfo(bucketName, objectName string, info minio.UploadInfo) ObjectInfo {
	return ObjectInfo{
		Bucket:         bucketName,
		Key:            objectName,
		Size:           info.Size,
		LastModified:   info.LastModified.Format(time.RFC3339),
		ETag:           info.ETag,
		ContentType:    "",  // UploadInfo doesn't contain content type
		Metadata:       nil, // UploadInfo doesn't contain metadata
		UserTags:       nil, // UploadInfo doesn't contain user tags
		VersionID:      info.VersionID,
		IsLatest:       true, // Assume latest for uploads
		IsDeleteMarker: false,
		StorageClass:   "",
	}
}

// convertObjectInfo converts minio.ObjectInfo to our ObjectInfo
func (c *Client) convertObjectInfo(info minio.ObjectInfo) ObjectInfo {
	return ObjectInfo{
		Bucket:         "", // ObjectInfo doesn't have bucket field
		Key:            info.Key,
		Size:           info.Size,
		LastModified:   info.LastModified.Format(time.RFC3339),
		ETag:           info.ETag,
		ContentType:    info.ContentType,
		Metadata:       info.UserMetadata,
		UserTags:       info.UserTags,
		VersionID:      info.VersionID,
		IsLatest:       info.IsLatest,
		IsDeleteMarker: info.IsDeleteMarker,
		StorageClass:   info.StorageClass,
	}
}

// convertListObjectInfo converts minio.ObjectInfo from list operations to our ObjectInfo
func (c *Client) convertListObjectInfo(info minio.ObjectInfo) ObjectInfo {
	return ObjectInfo{
		Bucket:         "", // ObjectInfo from list doesn't have bucket field
		Key:            info.Key,
		Size:           info.Size,
		LastModified:   info.LastModified.Format(time.RFC3339),
		ETag:           info.ETag,
		ContentType:    info.ContentType,
		Metadata:       info.UserMetadata,
		UserTags:       info.UserTags,
		VersionID:      info.VersionID,
		IsLatest:       info.IsLatest,
		IsDeleteMarker: info.IsDeleteMarker,
		StorageClass:   info.StorageClass,
	}
}

// GetContentType determines content type based on file extension
func GetContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".bmp":
		return "image/bmp"
	case ".tiff", ".tif":
		return "image/tiff"
	case ".ico":
		return "image/x-icon"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz":
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}
