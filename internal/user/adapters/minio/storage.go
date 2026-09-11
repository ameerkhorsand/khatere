package userminio

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	miniogo "github.com/minio/minio-go/v7"

	"github.com/google/uuid"
)

// publicURLTTL is how long a presigned read URL stays valid. Same
// window as internal/archive/adapters/minio.Storage.PublicURL — the
// frontend is expected to re-fetch, not cache, this URL.
const publicURLTTL = 15 * time.Minute

type Storage struct {
	client       *miniogo.Client // internal Docker-network client — Upload/Delete
	publicClient *miniogo.Client // public-host client — PublicURL signing only
	bucket       string
}

func New(client *miniogo.Client, publicClient *miniogo.Client, bucket string) *Storage {
	return &Storage{client: client, publicClient: publicClient, bucket: bucket}
}

// Upload stores the file under users/<accountID>/<uuid>-<filename>
// and returns that object key. contentType is trusted as already
// validated by the caller (see domain.ValidContentType) — this
// adapter does not re-check it.
func (s *Storage) Upload(ctx context.Context, accountID uuid.UUID, filename string, contentType string, content io.Reader, sizeBytes int64) (string, error) {
	key := fmt.Sprintf("users/%s/%s-%s", accountID, uuid.New(), filename)

	_, err := s.client.PutObject(ctx, s.bucket, key, content, sizeBytes, miniogo.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

func (s *Storage) Delete(ctx context.Context, storageKey string) error {
	return s.client.RemoveObject(ctx, s.bucket, storageKey, miniogo.RemoveObjectOptions{})
}

// PublicURL returns a time-limited URL the frontend can open
// directly to fetch the object. Signing happens locally — no
// network call is made — so publicClient's host never needs to be
// reachable from inside this process.
func (s *Storage) PublicURL(ctx context.Context, storageKey string) (string, error) {
	reqParams := url.Values{}
	presignedURL, err := s.publicClient.PresignedGetObject(ctx, s.bucket, storageKey, publicURLTTL, reqParams)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}
