package postgresql_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock" // Стандартная библиотека для моков SQL
	"github.com/KalessinD/gophermart/internal/models"
	"github.com/KalessinD/gophermart/internal/repositories/postgresql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLStorage_Ping(t *testing.T) {
	t.Run("successful ping", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectPing()

		storage := postgresql.NewSQLStorage(db)
		err = storage.Ping(context.Background())

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ping error", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectPing().WillReturnError(errors.New("connection refused"))

		storage := postgresql.NewSQLStorage(db)
		err = storage.Ping(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// тест для AddUser
func TestSQLStorage_AddUser(t *testing.T) {
	t.Run("successful add user", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		storage := postgresql.NewSQLStorage(db)

		// 1. Ожидаем начало транзакции
		mock.ExpectBegin()
		// 2. Ожидаем Exec внутри транзакции.
		// ВАЖНО: Если ваш запрос в user.go отличается, исправьте регулярное выражение.
		mock.ExpectExec("INSERT INTO ").
			WithArgs("testuser", sqlmock.AnyArg()). // Логин и любой хеш
			WillReturnResult(sqlmock.NewResult(1, 1))
		// 3. Ожидаем коммит
		mock.ExpectCommit()

		user := &models.User{
			Login: "testuser",
			Hash:  "hash",
		}

		err = storage.AddUser(context.Background(), user)

		// Если здесь паника с nil pointer, проверьте файл user.go
		// Вы должны вызывать: r.addUserTx(ctx, user, tx)
		// А не: r.addUserTx(ctx, user, nil)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("retry on connection error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		storage := postgresql.NewSQLStorage(db)

		// Симулируем ошибку, которую можно повторить (Code 08006 - Connection Failure)
		pgErr := &pgconn.PgError{
			Code:    "08006",
			Message: "connection error",
		}

		for i := 0; i < postgresql.RetryingAttempts; i++ {
			mock.ExpectBegin()
			mock.ExpectExec("INSERT").
				WillReturnError(pgErr)
			mock.ExpectRollback()
		}

		user := &models.User{Login: "test", Hash: "hash"}
		err = storage.AddUser(context.Background(), user)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
