package postgresql_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/KalessinD/gophermart/internal/models"
	"github.com/KalessinD/gophermart/internal/repositories/postgresql"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLStorage_GetOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := postgresql.NewSQLStorage(db)

	t.Run("successful get order", func(t *testing.T) {
		orderID := "12345"
		userID := "user_1"
		now := time.Now().Truncate(time.Microsecond)

		// QuerySelectOrder выбирает: id, user_id, status, accrual, uploaded_at, updated_at
		rows := sqlmock.NewRows([]string{"id", "user_id", "status", "accrual", "uploaded_at", "updated_at"}).
			AddRow(orderID, userID, models.OrderNewStatus, 100, now, now)

		mock.ExpectQuery("SELECT").
			WithArgs(orderID, userID).
			WillReturnRows(rows)

		order, err := storage.GetOrder(context.Background(), orderID, userID)
		require.NoError(t, err)
		assert.Equal(t, orderID, order.ID)
		assert.Equal(t, userID, order.UserID)
		assert.Equal(t, 100, order.Accrual)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("order not found", func(t *testing.T) {
		orderID := "not_found"
		userID := "user_2"

		mock.ExpectQuery("SELECT").
			WithArgs(orderID, userID).
			WillReturnError(sql.ErrNoRows)

		order, err := storage.GetOrder(context.Background(), orderID, userID)
		require.Error(t, err)
		assert.Nil(t, order)
		assert.ErrorIs(t, err, models.ErrOrderNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("retry on connection error then success", func(t *testing.T) {
		orderID := "retry_order"
		userID := "user_1"
		pgErr := &pgconn.PgError{Code: "08006"} // Connection failure

		// Первая попытка - ошибка соединения
		mock.ExpectQuery("SELECT").
			WithArgs(orderID, userID).
			WillReturnError(pgErr)

		// Вторая попытка - успех
		rows := sqlmock.NewRows([]string{"id", "user_id", "status", "accrual", "uploaded_at", "updated_at"}).
			AddRow(orderID, userID, models.OrderNewStatus, 0, time.Now(), time.Now())
		mock.ExpectQuery("SELECT").
			WithArgs(orderID, userID).
			WillReturnRows(rows)

		order, err := storage.GetOrder(context.Background(), orderID, userID)
		require.NoError(t, err)
		assert.Equal(t, orderID, order.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSQLStorage_ListOrders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := postgresql.NewSQLStorage(db)

	t.Run("successful list orders", func(t *testing.T) {
		userID := "user_3"
		now := time.Now().Truncate(time.Microsecond)

		rows := sqlmock.NewRows([]string{"id", "user_id", "status", "accrual", "uploaded_at", "updated_at"}).
			AddRow("order1", userID, models.OrderNewStatus, 0, now, now).
			AddRow("order2", userID, models.OrderProcessedStatus, 500, now, now)

		mock.ExpectQuery("SELECT").
			WithArgs(userID).
			WillReturnRows(rows)

		orders, err := storage.ListOrders(context.Background(), userID)
		require.NoError(t, err)
		require.Len(t, orders, 2)
		assert.Equal(t, "order1", orders[0].ID)
		assert.Equal(t, "order2", orders[1].ID)
		assert.Equal(t, 500, orders[1].Accrual)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list", func(t *testing.T) {
		userID := "user_empty"

		rows := sqlmock.NewRows([]string{"id", "user_id", "status", "accrual", "uploaded_at", "updated_at"})

		mock.ExpectQuery("SELECT").
			WithArgs(userID).
			WillReturnRows(rows)

		orders, err := storage.ListOrders(context.Background(), userID)
		require.NoError(t, err)
		assert.Empty(t, orders)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		userID := "user_err"

		mock.ExpectQuery("SELECT").
			WithArgs(userID).
			WillReturnError(errors.New("query failed"))

		orders, err := storage.ListOrders(context.Background(), userID)
		require.Error(t, err)
		assert.Nil(t, orders)
		assert.Contains(t, err.Error(), "query failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSQLStorage_AddOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := postgresql.NewSQLStorage(db)

	t.Run("successful add order", func(t *testing.T) {
		order := &models.Order{ID: "123", UserID: "user_4", Status: models.OrderNewStatus}

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT").
			WithArgs(order.ID, order.UserID, order.Status).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(order.ID))
		mock.ExpectCommit()

		err := storage.AddOrder(context.Background(), order)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("order already exists (unique violation)", func(t *testing.T) {
		order := &models.Order{ID: "123", UserID: "user_1", Status: models.OrderNewStatus}
		pgErr := &pgconn.PgError{Code: pgerrcode.UniqueViolation, Message: "duplicate key"}

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT").
			WithArgs(order.ID, order.UserID, order.Status).
			WillReturnError(pgErr)
		mock.ExpectRollback()

		err := storage.AddOrder(context.Background(), order)
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrOrderExists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database internal error", func(t *testing.T) {
		order := &models.Order{ID: "123", UserID: "user_1", Status: models.OrderNewStatus}

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT").
			WithArgs(order.ID, order.UserID, order.Status).
			WillReturnError(errors.New("some internal error"))
		mock.ExpectRollback()

		err := storage.AddOrder(context.Background(), order)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "some internal error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("retry logic on insert", func(t *testing.T) {
		order := &models.Order{ID: "retry", UserID: "user_1", Status: models.OrderNewStatus}
		pgErr := &pgconn.PgError{Code: pgerrcode.ConnectionFailure}

		// Первая попытка - ошибка соединения (ретрается в withTxRetry)
		mock.ExpectBegin()
		mock.ExpectQuery("INSERT").
			WithArgs(order.ID, order.UserID, order.Status).
			WillReturnError(pgErr)
		mock.ExpectRollback()

		// Вторая попытка - успех
		mock.ExpectBegin()
		mock.ExpectQuery("INSERT").
			WithArgs(order.ID, order.UserID, order.Status).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(order.ID))
		mock.ExpectCommit()

		err := storage.AddOrder(context.Background(), order)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
