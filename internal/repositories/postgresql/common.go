package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	model "github.com/KalessinD/gophermart/internal/models"
	"github.com/KalessinD/gophermart/internal/services/db/pgerrors"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	RetryingAttempts  = 3
	RetryingDelay     = 100 * time.Millisecond
	RetryingDelayStep = 200 * time.Millisecond

	PsqlGophermartSchema = "gophermart"
)

type (
	txWrapperFunc func(tx *sql.Tx) error
	wrapperFunc   func(ctx context.Context) (*sql.Row, error)

	SQLStorage struct {
		db *sql.DB
	}

	SQLStorageInterface interface {
		GetUser(ctx context.Context, login string) (*model.User, error)
		AddUser(ctx context.Context, user *model.User) error
		Ping(ctx context.Context) error
	}
)

/*
Конструктор структуры для работы с БД
*/
func NewSQLStorage(psql *sql.DB) SQLStorageInterface {
	return &SQLStorage{db: psql}
}

/*
Пингуем сервер БД
*/
func (r *SQLStorage) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *SQLStorage) withRetry(ctx context.Context, action wrapperFunc) (*sql.Row, error) {
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
