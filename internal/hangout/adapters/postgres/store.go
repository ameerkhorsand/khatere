package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// executor is the subset of *pgxpool.Pool and pgx.Tx that the
// repositories need. Both types satisfy it, so a repository method
// can run against a bare connection or against an open transaction
// without knowing which one it got.
type executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txContextKey struct{}

// Store implements domain.Transactor and is the single owner of the
// connection pool. Build one Store per process and pass it to the
// three repositories below, and to any use case that needs
// WithinTransaction (currently CreateHangoutUseCase).
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// WithinTransaction opens a transaction, runs fn with a context that
// carries it, and commits on success or rolls back on error/panic.
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

// dbFrom returns the active transaction if ctx carries one (set by
// WithinTransaction above), otherwise the plain pool.
func dbFrom(ctx context.Context, pool *pgxpool.Pool) executor {
	if tx, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}
