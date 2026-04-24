package repositories

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

	PsqlUsersSchema = "gophermart"
	PsqlUserTable   = "users"

	QueryInsertUser = `
		INSERT INTO "` + PsqlUsersSchema + `"."` + PsqlUserTable + `" AS t (login, hash, salt)
        VALUES($1, $2, $3)
		`

	QuerySelectUser = `SELECT login, hash, salt, created_at FROM "` + PsqlUsersSchema + `"."` + PsqlUserTable + `" WHERE login = $1`
)

type (
	txWrapperFunc func(tx *sql.Tx) error
	wrapperFunc   func(ctx context.Context) (any, error)

	SQLStorage struct {
		db *sql.DB
	}

	SQLStorageInterface interface {
		GetUser(ctx context.Context, login string) (*model.User, error)
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

func (r *SQLStorage) GetUser(ctx context.Context, login string) (*model.User, error) {
	obj, err := r.withRetry(ctx, func(ctx context.Context) (any, error) {
		row := r.db.QueryRowContext(ctx, QuerySelectUser, login)
		if row.Err() != nil {
			return nil, row.Err()
		}
		return row, nil
	})
	if err != nil {
		return nil, err
	}

	row, ok := obj.(*sql.Row)
	if !ok {
		return nil, errors.New("unexpected type")
	}

	user := &model.User{}
	err = row.Scan(&user.Login, &user.Hash, &user.Salt, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *SQLStorage) AddUser(ctx context.Context, user *model.User) error {
	return r.withTxRetry(ctx, func(tx *sql.Tx) error {
		if err := r.addUserTx(ctx, tx, user); err != nil {
			return err
		}
		return nil
	})
}

func (r *SQLStorage) addUserTx(ctx context.Context, tx *sql.Tx, user *model.User) error {
	_, err := tx.ExecContext(ctx, QueryInsertUser, user.Login, user.Hash, user.Salt)
	return err
}

func (r *SQLStorage) withRetry(ctx context.Context, action wrapperFunc) (any, error) {
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
