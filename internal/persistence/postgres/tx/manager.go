package tx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Manager interface {
	WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error
}

type manager struct {
	db *pgxpool.Pool
}

func NewManager(db *pgxpool.Pool) Manager {
	return &manager{
		db: db,
	}
}

func (m *manager) WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error {

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
