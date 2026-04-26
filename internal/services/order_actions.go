//go:generate mockgen -source=user_common_actions.go -destination=mocks/mock_user_common_actions.go -package=mocks
package services

import (
	"context"

	"github.com/KalessinD/gophermart/internal/middleware"
	"github.com/KalessinD/gophermart/internal/models"
	repository "github.com/KalessinD/gophermart/internal/repositories/postgresql"
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
Выполняет вход в систему.
Может вернуть ошибку, если таковая произошла
*/
func (s *OrderActions) Store(ctx context.Context, idStr string) error {
	claims := middleware.GetClaims(ctx)
	order, err := models.NewOrder(idStr, claims.ID, models.OrderNewStatus)
	if err != nil {
		return err
	}

	if err := order.Validate(); err != nil {
		return err
	}

	// return s.db.AddOrder(order)

	return nil
}
