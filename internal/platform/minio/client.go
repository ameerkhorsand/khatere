package minio

import (
	"context"
	"net/url"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// defaultRegion is MinIO's own default region. Setting it explicitly
// stops the minio-go client from calling GetBucketLocation the first
// time it needs to sign a request (including presigned URLs). That
// call would otherwise go out to the client's own configured
// endpoint — for minioPublicClient, that is MINIO_PUBLIC_URL (e.g.
// "http://localhost:9000"), which is not reachable from inside the
// api container, since "localhost" there means the container itself,
// not the minio container or the host machine.
const defaultRegion = "us-east-1"

// NewClient builds a MinIO client for the given endpoint (e.g. "minio:9000").
// Secure is false because traffic stays inside the Docker network.
func NewClient(endpoint, accessKey, secretKey string) (*miniogo.Client, error) {
	return miniogo.New(endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
		Region: defaultRegion,
	})
}

// NewClientWithOptions is NewClient with an explicit Secure flag, for
// hosts outside the Docker network that may use TLS (https).
func NewClientWithOptions(endpoint, accessKey, secretKey string, secure bool) (*miniogo.Client, error) {
	return miniogo.New(endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
		Region: defaultRegion,
	})
}

// ParsePublicURL splits a full URL (e.g. "http://localhost:9000")
// into the bare host MinIO's client wants, and whether it's https.
func ParsePublicURL(rawURL string) (host string, secure bool, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false, err
	}
	return u.Host, u.Scheme == "https", nil
}

// Ping checks that MinIO answers, by listing buckets.
func Ping(ctx context.Context, client *miniogo.Client) error {
	_, err := client.ListBuckets(ctx)
	return err
}
