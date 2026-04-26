package postgresql

import (
	"context"
	"database/sql"
	"errors"

	model "github.com/KalessinD/gophermart/internal/models"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	PsqlUsersSchema = "gophermart"
	PsqlUserTable   = "users"

	QueryInsertUser = `
		INSERT INTO "` + PsqlUsersSchema + `"."` + PsqlUserTable + `" AS t (login, hash)
        VALUES($1, $2)
		`

	QuerySelectUser = `SELECT id, login, hash, version, created_at FROM "` + PsqlUsersSchema + `"."` + PsqlUserTable + `" WHERE login = $1`
)

type (
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
	row, err := withRetry(ctx, func(ctx context.Context) (*sql.Row, error) {
		row := r.db.QueryRowContext(ctx, QuerySelectUser, login)
		if row.Err() != nil {
			return nil, row.Err()
		}
		return row, nil
	})
	if err != nil {
		return nil, err
	}

	user := &model.User{}
	err = row.Scan(&user.ID, &user.Login, &user.Hash, &user.Version, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *SQLStorage) AddUser(ctx context.Context, user *model.User) error {
	return withTxRetry(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.addUserTx(ctx, tx, user); err != nil {
			return err
		}
		return nil
	})
}

func (r *SQLStorage) addUserTx(ctx context.Context, tx *sql.Tx, user *model.User) error {
	_, err := tx.ExecContext(ctx, QueryInsertUser, user.Login, user.Hash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique constraint violation (login or id)
				return model.ErrUserExists
			}
		}
		return err
	}
	return nil
}
