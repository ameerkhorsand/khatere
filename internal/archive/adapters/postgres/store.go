package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// executor is the subset of *pgxpool.Pool and pgx.Tx the archive
// repositories need. Same pattern as internal/hangout/adapters/postgres.
type executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txContextKey struct{}

// Store implements domain.Transactor for the archive module. Build
// one per process and pass it to the repositories below. Used to
// wrap MarkArchiveDeletedUseCase and CheckAndPurgeArchiveUseCase in
// one transaction from the HTTP handler (Step 10), so a mark can
// never be counted twice by a concurrent request.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}

	txCtx := context.WithValue(ctx, txContextKey{}, tx)

	if err := fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

func dbFrom(ctx context.Context, pool *pgxpool.Pool) executor {
	if tx, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}
