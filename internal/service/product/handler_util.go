// Package product provides HTTP handler utilities for product services.
//
// This file contains shared utilities for enriching product responses with
// presigned URLs generated at the handler layer.
package product

import (
	"context"
	"log"

	"go-shopping-poc/internal/platform/storage/minio"
)

// ImageURLGenerator handles presigned URL generation for handlers
type ImageURLGenerator struct {
	objectStorage minio.ObjectStorage
	bucket        string
}

// NewImageURLGenerator creates a new URL generator
func NewImageURLGenerator(objectStorage minio.ObjectStorage, bucket string) *ImageURLGenerator {
	return &ImageURLGenerator{
		objectStorage: objectStorage,
		bucket:        bucket,
	}
}

// EnrichProductWithImageURLs generates presigned URLs for all product images
func (g *ImageURLGenerator) EnrichProductWithImageURLs(ctx context.Context, product *Product) {
	if product == nil || g.objectStorage == nil {
		return
	}

	for i := range product.Images {
		if product.Images[i].MinioObjectName != "" {
			url, err := g.objectStorage.PresignedGetObject(ctx, g.bucket, product.Images[i].MinioObjectName, 3600)
			if err != nil {
				log.Printf("[WARN] Failed to generate presigned URL for image %d: %v", product.Images[i].ID, err)
				continue
			}
			product.Images[i].ImageURL = url
		}
	}
}

// EnrichProductsWithImageURLs generates presigned URLs for a slice of products
func (g *ImageURLGenerator) EnrichProductsWithImageURLs(ctx context.Context, products []*Product) {
	for _, product := range products {
		g.EnrichProductWithImageURLs(ctx, product)
	}
}

// GenerateImageURL generates a single presigned URL for an image
func (g *ImageURLGenerator) GenerateImageURL(ctx context.Context, minioObjectName string) (string, error) {
	if minioObjectName == "" {
		return "", nil
	}
	if g.objectStorage == nil {
		return "", nil
	}
	return g.objectStorage.PresignedGetObject(ctx, g.bucket, minioObjectName, 3600)
}
