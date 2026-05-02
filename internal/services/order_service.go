//go:generate mockgen -source=order_service.go -destination=mocks/mock_order_service.go -package=mocks
package services

import (
	"context"
	"errors"

	"github.com/KalessinD/gophermart/internal/clients"
	"github.com/KalessinD/gophermart/internal/middleware"
	"github.com/KalessinD/gophermart/internal/models"
	"github.com/KalessinD/gophermart/internal/processors"
	repository "github.com/KalessinD/gophermart/internal/repositories/postgresql"
	"github.com/KalessinD/gophermart/internal/services/alg"
)

type (
	/*
		Объект службы для работы с заказами
	*/
	OrderActions struct {
		db            repository.SQLStorageInterface
		accrualClient clients.AccrualClienttInterface
	}

	/*
		Интерфейс оОъекта службы заказов
	*/
	OrderActionsInterface interface {
		Store(ctx context.Context, orderID string) error
		List(ctx context.Context) (models.OrdersList, error)
		// GetOrderAccrual(ctx context.Context, order *models.Order) (*models.AccrualResponse, error)
		ProcessAccrualTask(ctx context.Context, task *processors.Task) error
	}
)

/*
Конструктор службы для операций с заказами
*/
func NewOrderActions(db repository.SQLStorageInterface, client clients.AccrualClienttInterface) OrderActionsInterface {
	return &OrderActions{
		db:            db,
		accrualClient: client,
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

func (s *OrderActions) ProcessAccrualTask(ctx context.Context, task *processors.Task) error {
	resp, err := s.accrualClient.GetOrderAccrual(ctx, task.ID)
	if err != nil {
		if errors.Is(err, clients.ErrServiceIsBusy) {
			// тут можно обрабатывать 429 Too Many Requests
			_ = 1
		}
		return err
	}

	_ = resp

	//	err = s.db.UpdateOrderStatus(ctx, order.ID, resp.Status, resp.Accrual)
	//	if err != nil {
	//		return err
	//	}

	return nil
}
