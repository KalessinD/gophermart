package repositories

import (
	"context"
	"database/sql"
	"time"

	model "github.com/KalessinD/gophermart/internal/models"
)

const (
	RetryingAttempts  = 3
	RetryingDelay     = 100 * time.Millisecond
	RetryingDelayStep = 200 * time.Millisecond

	PsqlMetricsSchema = "gophermart"
	PsqlUserTable     = "users"
)

type (
	// txWrapperFunc func(tx *sql.Tx) error

	SQLStorage struct {
		db *sql.DB
	}

	SQLStorageInterface interface {
		GetUser(ctx context.Context) *model.User
		AddUser(ctx context.Context, user *model.User) error
		Ping(ctx context.Context) error
	}
)

func NewSQLStorage(psql *sql.DB) SQLStorageInterface {
	return &SQLStorage{db: psql}
}

func (r *SQLStorage) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *SQLStorage) GetUser(_ context.Context) *model.User {
	return nil
}

func (r *SQLStorage) AddUser(_ context.Context, _ *model.User) error {
	return nil
}

/*
func (r *SQLStorage) withTxRetry(ctx context.Context, action txWrapperFunc) error {
	var err error

	for attempts := 0; attempts < RetryingAttempts; attempts++ {
		tx, err := r.db.BeginTx(ctx, nil)
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
*/
