package minio

import (
	"context"
	"io"
)

// ObjectStorage defines the interface for object storage operations
type ObjectStorage interface {
	// Bucket operations
	CreateBucket(ctx context.Context, bucketName string) error
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	DeleteBucket(ctx context.Context, bucketName string) error

	// Object operations
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts PutObjectOptions) (ObjectInfo, error)
	GetObject(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error)
	StatObject(ctx context.Context, bucketName, objectName string) (ObjectInfo, error)
	RemoveObject(ctx context.Context, bucketName, objectName string) error
	CopyObject(ctx context.Context, dstBucket, dstObject, srcBucket, srcObject string, opts CopyObjectOptions) error

	// File operations (convenience methods)
	FPutObject(ctx context.Context, bucketName, objectName, filePath string, opts PutObjectOptions) (ObjectInfo, error)
	FGetObject(ctx context.Context, bucketName, objectName, filePath string) error

	// List operations
	ListObjects(ctx context.Context, bucketName string, opts ListObjectsOptions) <-chan ObjectInfo

	// Presigned URL operations
	PresignedGetObject(ctx context.Context, bucketName, objectName string, expirySeconds int64) (string, error)
	PresignedPutObject(ctx context.Context, bucketName, objectName string, expirySeconds int64) (string, error)
}

// PutObjectOptions contains options for PutObject operations
type PutObjectOptions struct {
	ContentType        string
	ContentEncoding    string
	ContentLanguage    string
	ContentDisposition string
	CacheControl       string
	Metadata           map[string]string
	UserTags           map[string]string
	StorageClass       string
	Retention          ObjectRetention
	LegalHold          bool
}

// CopyObjectOptions contains options for CopyObject operations
type CopyObjectOptions struct {
	Metadata        map[string]string
	UserTags        map[string]string
	ReplaceMetadata bool
	StorageClass    string
	Retention       ObjectRetention
	LegalHold       bool
}

// ListObjectsOptions contains options for ListObjects operations
type ListObjectsOptions struct {
	Prefix       string
	StartAfter   string
	Recursive    bool
	MaxKeys      int
	WithMetadata bool
	WithVersions bool
}

// ObjectInfo contains information about an object
type ObjectInfo struct {
	Bucket         string
	Key            string
	Size           int64
	LastModified   string
	ETag           string
	ContentType    string
	Metadata       map[string]string
	UserTags       map[string]string
	VersionID      string
	IsLatest       bool
	IsDeleteMarker bool
	StorageClass   string
}

// ObjectRetention contains retention settings for an object
type ObjectRetention struct {
	Mode            RetentionMode
	RetainUntilDate string
}

// RetentionMode represents retention modes
type RetentionMode string

const (
	RetentionModeNone       RetentionMode = ""
	RetentionModeGovernance RetentionMode = "GOVERNANCE"
	RetentionModeCompliance RetentionMode = "COMPLIANCE"
)
