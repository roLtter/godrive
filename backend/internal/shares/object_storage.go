package shares

import (
	"context"
	"net/url"
)

// ObjectStorage provides presigned download links for shared files.
type ObjectStorage interface {
	PresignedGetURL(ctx context.Context, objectName string) (*url.URL, error)
}
