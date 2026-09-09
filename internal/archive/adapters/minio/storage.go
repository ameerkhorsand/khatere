// Package archiveminio implements domain.MediaStorage using the
// shared MinIO client from internal/platform/minio.
package archiveminio

import (
	"context"
	"fmt"
	"io"

	miniogo "github.com/minio/minio-go/v7"

	"github.com/google/uuid"

	"github.com/bLorax/khatere-backend/internal/archive/domain"
)

type Storage struct {
	client *miniogo.Client
	bucket string
}

func New(client *miniogo.Client, bucket string) *Storage {
	return &Storage{client: client, bucket: bucket}
}

// Upload stores the file under archives/<archiveID>/<uuid>-<filename>
// and returns that object key.
func (s *Storage) Upload(ctx context.Context, archiveID uuid.UUID, mediaType domain.MediaType, filename string, content io.Reader, sizeBytes int64) (string, error) {
	key := fmt.Sprintf("archives/%s/%s-%s", archiveID, uuid.New(), filename)

	_, err := s.client.PutObject(ctx, s.bucket, key, content, sizeBytes, miniogo.PutObjectOptions{
		ContentType: contentTypeFor(mediaType),
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

func (s *Storage) Delete(ctx context.Context, storageKey string) error {
	return s.client.RemoveObject(ctx, s.bucket, storageKey, miniogo.RemoveObjectOptions{})
}

func contentTypeFor(mediaType domain.MediaType) string {
	switch mediaType {
	case domain.MediaTypePhoto:
		return "image/jpeg"
	case domain.MediaTypeGIF:
		return "image/gif"
	case domain.MediaTypeVideo:
		return "video/mp4"
	case domain.MediaTypeAudio:
		return "audio/mpeg"
	default:
		return "application/octet-stream"
	}
}
