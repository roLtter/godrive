package minio

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client wraps MinIO SDK and project-level defaults.
type Client struct {
	sdkClient  *minio.Client
	bucketName string
	presignTTL time.Duration
}

// New creates MinIO client, validates connectivity and ensures bucket exists.
func New(ctx context.Context, endpoint, accessKey, secretKey, bucketName string, presignTTLMinutes int) (*Client, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("minio client: endpoint is required")
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("minio client: parse endpoint: %w", err)
	}

	useSSL := strings.EqualFold(parsed.Scheme, "https")
	baseEndpoint := parsed.Host
	if baseEndpoint == "" {
		baseEndpoint = parsed.Path
	}

	sdkClient, err := minio.New(baseEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: create sdk client: %w", err)
	}

	exists, err := sdkClient.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("minio client: check bucket exists: %w", err)
	}
	if !exists {
		if err := sdkClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio client: create bucket: %w", err)
		}
	}

	return &Client{
		sdkClient:  sdkClient,
		bucketName: bucketName,
		presignTTL: time.Duration(presignTTLMinutes) * time.Minute,
	}, nil
}

// PresignedPutURL returns a temporary URL for direct object upload.
func (c *Client) PresignedPutURL(ctx context.Context, objectName string) (*url.URL, error) {
	url, err := c.sdkClient.PresignedPutObject(ctx, c.bucketName, objectName, c.presignTTL)
	if err != nil {
		return nil, fmt.Errorf("minio client: create presigned put url: %w", err)
	}
	return url, nil
}

// PresignedGetURL returns a temporary URL for direct object download.
func (c *Client) PresignedGetURL(ctx context.Context, objectName string) (*url.URL, error) {
	url, err := c.sdkClient.PresignedGetObject(ctx, c.bucketName, objectName, c.presignTTL, nil)
	if err != nil {
		return nil, fmt.Errorf("minio client: create presigned get url: %w", err)
	}
	return url, nil
}

// PutObject uploads stream into MinIO bucket.
func (c *Client) PutObject(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	_, err := c.sdkClient.PutObject(ctx, c.bucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("minio client: put object: %w", err)
	}
	return nil
}
