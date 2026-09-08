package minio

import (
	"context"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// NewClient builds a MinIO client for the given endpoint (e.g. "minio:9000").
// Secure is false because traffic stays inside the Docker network.
func NewClient(endpoint, accessKey, secretKey string) (*miniogo.Client, error) {
	return miniogo.New(endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
}

// Ping checks that MinIO answers, by listing buckets.
func Ping(ctx context.Context, client *miniogo.Client) error {
	_, err := client.ListBuckets(ctx)
	return err
}
