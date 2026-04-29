package postgresql

import (
	"context"
	"database/sql"
	"errors"

	model "github.com/KalessinD/gophermart/internal/models"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	PsqlOrderTable = `"` + PsqlGophermartSchema + `"."orders"`

	QueryInsertOrder = `INSERT INTO ` + PsqlOrderTable + ` AS t (id, user_id, status) VALUES($1, $2, $3) RETURNING id`

	QuerySelectOrder = `SELECT id, user_id, status, accrual, uploaded_at, updated_at FROM ` + PsqlOrderTable + ` WHERE id = $1 AND user_id = $2`
)

func (r *SQLStorage) GetOrder(ctx context.Context, orderID string) (*model.Order, error) {
	order := &model.Order{}

	_, err := r.withRetry(ctx, func(ctx context.Context) (*sql.Row, error) {
		row := r.db.QueryRowContext(ctx, QuerySelectOrder, orderID)
		if err := row.Scan(&order.ID, &order.UserID, &order.Status, &order.Accrual, &order.UpdatedAt, &order.UpdatedAt); err != nil {
			return nil, err
		}
		return row, nil
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (r *SQLStorage) AddOrder(ctx context.Context, order *model.Order) error {
	return r.withTxRetry(ctx, func(tx *sql.Tx) error {
		if err := r.addOrderTx(ctx, tx, order); err != nil {
			return err
		}
		return nil
	})
}

func (r *SQLStorage) addOrderTx(ctx context.Context, tx *sql.Tx, order *model.Order) error {
	err := tx.QueryRowContext(ctx, QueryInsertOrder, order.ID, order.UserID, order.Status).Scan(&order.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return model.ErrOrderExists
			}
		}
		return err
	}
	return nil
}
