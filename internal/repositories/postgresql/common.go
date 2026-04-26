package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/KalessinD/gophermart/internal/services/db/pgerrors"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	RetryingAttempts  = 3
	RetryingDelay     = 100 * time.Millisecond
	RetryingDelayStep = 200 * time.Millisecond
)

type (
	txWrapperFunc func(tx *sql.Tx) error
	wrapperFunc   func(ctx context.Context) (*sql.Row, error)
)

func withRetry(ctx context.Context, action wrapperFunc) (*sql.Row, error) {
	var err error

	for attempts := 0; attempts < RetryingAttempts; attempts++ {
		obj, err := action(ctx)

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgerrors.ClassifyPgError(pgErr) == pgerrors.Retriable {
				time.Sleep(time.Millisecond * 100)
				continue
			}
		}

		return obj, err
	}

	return nil, err
}

func withTxRetry(ctx context.Context, db *sql.DB, action txWrapperFunc) error {
	var err error

	for attempts := 0; attempts < RetryingAttempts; attempts++ {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		err = action(tx)
		if err == nil {
			return tx.Commit()
		}

		_ = tx.Rollback()

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgerrors.ClassifyPgError(pgErr) == pgerrors.Retriable {
				time.Sleep(time.Millisecond * 100)
				continue
			}
		}

		return err
	}

	return err
}
