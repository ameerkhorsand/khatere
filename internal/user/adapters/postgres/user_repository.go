package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/bLorax/khatere-backend/internal/user/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	meta, err := json.Marshal(u.Metadata)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO users (account_id, display_name, handle, bio, profile_picture_key, metadata, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, u.AccountID, u.DisplayName, u.Handle, u.Bio, u.ProfilePictureKey, meta, u.Version, u.CreatedAt, u.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "users_handle_key" {
				return domain.ErrHandleTaken
			}
		}
		return err
	}
	return nil
}

func (r *UserRepository) FindByAccountID(ctx context.Context, accountID uuid.UUID) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT account_id, display_name, handle, bio, profile_picture_key, metadata, version, created_at, updated_at, deleted_at
		FROM users WHERE account_id = $1
	`, accountID)

	var u domain.User
	var metaBytes []byte
	err := row.Scan(&u.AccountID, &u.DisplayName, &u.Handle, &u.Bio, &u.ProfilePictureKey, &metaBytes,
		&u.Version, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(metaBytes, &u.Metadata); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByHandle(ctx context.Context, handle string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT account_id, display_name, handle, bio, profile_picture_key, metadata, version, created_at, updated_at, deleted_at
		FROM users
		WHERE handle = $1
	`, handle)

	var u domain.User
	var metaBytes []byte

	err := row.Scan(
		&u.AccountID,
		&u.DisplayName,
		&u.Handle,
		&u.Bio,
		&u.ProfilePictureKey,
		&metaBytes,
		&u.Version,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	if err := json.Unmarshal(metaBytes, &u.Metadata); err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	meta, err := json.Marshal(u.Metadata)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE users
		SET display_name = $1, bio = $2, profile_picture_key = $3, metadata = $4, deleted_at = $5
		WHERE account_id = $6 AND version = $7
	`, u.DisplayName, u.Bio, u.ProfilePictureKey, meta, u.DeletedAt, u.AccountID, u.Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}
