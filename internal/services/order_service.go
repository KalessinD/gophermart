//go:generate mockgen -source=order_service.go -destination=mocks/mock_order_service.go -package=mocks
package services

import (
	"context"
	"errors"

	"github.com/KalessinD/gophermart/internal/middleware"
	"github.com/KalessinD/gophermart/internal/models"
	repository "github.com/KalessinD/gophermart/internal/repositories/postgresql"
	"github.com/KalessinD/gophermart/internal/services/alg"
)

type (
	/*
		Объект службы для работы с заказами
	*/
	OrderActions struct {
		db repository.SQLStorageInterface
	}

	/*
		Интерфейс оОъекта службы заказов
	*/
	OrderActionsInterface interface {
		Store(ctx context.Context, orderID string) error
		List(ctx context.Context) (models.OrdersList, error)
	}

	AccrualProvider interface {
		GetOrderAccrual(ctx context.Context, order *models.Order) (*models.AccrualResponse, error)
	}
)

/*
Конструктор службы для операций с заказами
*/
func NewOrderActions(db repository.SQLStorageInterface) OrderActionsInterface {
	return &OrderActions{
		db: db,
	}
}

/*
Сохраняет заказ в БД и отправляет его на обработку в Accrual
*/
func (s *OrderActions) Store(ctx context.Context, idStr string) error {
	if !alg.IsValidLuhn(idStr) {
		return models.ErrOrderWrongFormat
	}

	claims := middleware.GetClaims(ctx)
	order, err := models.NewOrder(idStr, claims.UserID, models.OrderNewStatus)
	if err != nil {
		return err
	}

	if err := order.Validate(); err != nil {
		return err
	}

	err = s.db.AddOrder(ctx, order)
	if err != nil {
		if !errors.Is(err, models.ErrOrderExists) {
			return err
		}

		_, err = s.db.GetOrder(ctx, order.ID, order.UserID)

		switch {
		case errors.Is(err, models.ErrOrderNotFound):
			return models.ErrOrderBelongsToOtherUser
		case err == nil:
			return models.ErrOrderExists
		default:
			return err
		}
	}

	return nil
}

func (s *OrderActions) List(ctx context.Context) (models.OrdersList, error) {
	claims := middleware.GetClaims(ctx)
	orders, err := s.db.ListOrders(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	return orders, nil
}
