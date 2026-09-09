package postgres

import (
	"context"
	"errors"

	"github.com/bLorax/khatere-backend/internal/circle/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConnectionRepository struct {
	pool *pgxpool.Pool
}

func NewConnectionRepository(pool *pgxpool.Pool) *ConnectionRepository {
	return &ConnectionRepository{pool: pool}
}

func (r *ConnectionRepository) CreateRequest(
	ctx context.Context,
	requesterID, addresseeID uuid.UUID,
) (*domain.Connection, error) {
	if requesterID == addresseeID {
		return nil, domain.ErrCannotConnectSelf
	}

	// Check the existing relationship first. The database also protects
	// against races with the unique canonical pair constraint.
	var existing domain.Connection

	err := r.pool.QueryRow(ctx, `
		SELECT id, requester_id, addressee_id, status, version, created_at, updated_at
		FROM circle_connections
		WHERE canonical_low = LEAST($1::uuid, $2::uuid)
		  AND canonical_high = GREATEST($1::uuid, $2::uuid)
	`, requesterID, addresseeID).Scan(
		&existing.ID,
		&existing.RequesterID,
		&existing.AddresseeID,
		&existing.Status,
		&existing.Version,
		&existing.CreatedAt,
		&existing.UpdatedAt,
	)

	if err == nil {
		if existing.Status == domain.ConnectionStatusAccepted {
			return nil, domain.ErrAlreadyConnected
		}

		return nil, domain.ErrRequestAlreadySent
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	var connection domain.Connection

	err = r.pool.QueryRow(ctx, `
		INSERT INTO circle_connections (
			requester_id,
			addressee_id,
			status
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			requester_id,
			addressee_id,
			status,
			version,
			created_at,
			updated_at
	`, requesterID, addresseeID, domain.ConnectionStatusPending).Scan(
		&connection.ID,
		&connection.RequesterID,
		&connection.AddresseeID,
		&connection.Status,
		&connection.Version,
		&connection.CreatedAt,
		&connection.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrRequestAlreadySent
		}

		return nil, err
	}

	return &connection, nil
}

func (r *ConnectionRepository) Accept(
	ctx context.Context,
	requestID, addresseeID uuid.UUID,
) (*domain.Connection, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		connection domain.Connection
	)

	// Lock the request row for the duration of the transaction.
	//
	// This is important because circle_blocks has a trigger that can
	// delete a connection. By locking the connection first, a concurrent
	// block operation cannot delete it between our block check and accept.
	err = tx.QueryRow(ctx, `
		SELECT
			id,
			requester_id,
			addressee_id,
			status,
			version,
			created_at,
			updated_at
		FROM circle_connections
		WHERE id = $1
		  AND addressee_id = $2
		FOR UPDATE
	`,
		requestID,
		addresseeID,
	).Scan(
		&connection.ID,
		&connection.RequesterID,
		&connection.AddresseeID,
		&connection.Status,
		&connection.Version,
		&connection.CreatedAt,
		&connection.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrRequestNotFound
	}
	if err != nil {
		return nil, err
	}

	if connection.Status != domain.ConnectionStatusPending {
		if connection.Status == domain.ConnectionStatusAccepted {
			return nil, domain.ErrAlreadyConnected
		}

		return nil, domain.ErrRequestNotFound
	}

	// Final, atomic block check.
	//
	// Blocking is directional, so both directions matter:
	//   requester -> addressee
	//   addressee -> requester
	var blocked bool

	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM circle_blocks
			WHERE (blocker_id = $1 AND blocked_id = $2)
			   OR (blocker_id = $2 AND blocked_id = $1)
		)
	`,
		connection.RequesterID,
		connection.AddresseeID,
	).Scan(&blocked)

	if err != nil {
		return nil, err
	}

	if blocked {
		return nil, domain.ErrBlocked
	}

	err = tx.QueryRow(ctx, `
		UPDATE circle_connections
		SET status = $1
		WHERE id = $2
		  AND addressee_id = $3
		  AND status = $4
		RETURNING
			id,
			requester_id,
			addressee_id,
			status,
			version,
			created_at,
			updated_at
	`,
		domain.ConnectionStatusAccepted,
		requestID,
		addresseeID,
		domain.ConnectionStatusPending,
	).Scan(
		&connection.ID,
		&connection.RequesterID,
		&connection.AddresseeID,
		&connection.Status,
		&connection.Version,
		&connection.CreatedAt,
		&connection.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrRequestNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &connection, nil
}

func (r *ConnectionRepository) Decline(
	ctx context.Context,
	requestID, addresseeID uuid.UUID,
) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM circle_connections
		WHERE id = $1
		  AND addressee_id = $2
		  AND status = $3
	`,
		requestID,
		addresseeID,
		domain.ConnectionStatusPending,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrRequestNotFound
	}

	return nil
}

func (r *ConnectionRepository) Sever(
	ctx context.Context,
	connectionID, userID uuid.UUID,
) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM circle_connections
		WHERE id = $1
		  AND status = $2
		  AND (requester_id = $3 OR addressee_id = $3)
	`,
		connectionID,
		domain.ConnectionStatusAccepted,
		userID,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrConnectionNotFound
	}

	return nil
}

func (r *ConnectionRepository) ListCircle(
	ctx context.Context,
	userID uuid.UUID,
) ([]domain.Connection, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			requester_id,
			addressee_id,
			status,
			version,
			created_at,
			updated_at
		FROM circle_connections
		WHERE status = $1
		  AND (requester_id = $2 OR addressee_id = $2)
		ORDER BY updated_at DESC, id DESC
	`,
		domain.ConnectionStatusAccepted,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanConnections(rows)
}

func (r *ConnectionRepository) ListIncomingPending(
	ctx context.Context,
	userID uuid.UUID,
) ([]domain.Connection, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			requester_id,
			addressee_id,
			status,
			version,
			created_at,
			updated_at
		FROM circle_connections
		WHERE addressee_id = $1
		  AND status = $2
		ORDER BY created_at DESC, id DESC
	`,
		userID,
		domain.ConnectionStatusPending,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanConnections(rows)
}

func (r *ConnectionRepository) ListOutgoingPending(
	ctx context.Context,
	userID uuid.UUID,
) ([]domain.Connection, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			requester_id,
			addressee_id,
			status,
			version,
			created_at,
			updated_at
		FROM circle_connections
		WHERE requester_id = $1
		  AND status = $2
		ORDER BY created_at DESC, id DESC
	`,
		userID,
		domain.ConnectionStatusPending,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanConnections(rows)
}

func scanConnections(rows pgx.Rows) ([]domain.Connection, error) {
	connections := make([]domain.Connection, 0)

	for rows.Next() {
		var connection domain.Connection

		if err := rows.Scan(
			&connection.ID,
			&connection.RequesterID,
			&connection.AddresseeID,
			&connection.Status,
			&connection.Version,
			&connection.CreatedAt,
			&connection.UpdatedAt,
		); err != nil {
			return nil, err
		}

		connections = append(connections, connection)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return connections, nil
}
