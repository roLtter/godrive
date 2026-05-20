package files

import (
	"context"
	"io"
	"net/url"
)

// ObjectStorage abstracts object storage for file handlers (MinIO in production).
type ObjectStorage interface {
	PutObject(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error
	PresignedGetURL(ctx context.Context, objectName string) (*url.URL, error)
	RemoveObject(ctx context.Context, objectName string) error
}
