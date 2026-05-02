//go:generate mockgen -source=balance_service.go -destination=mocks/mock_balance_service.go -package=mocks
package services

import (
	"context"

	repository "github.com/KalessinD/gophermart/internal/repositories/postgresql"
)

type (
	/*
		Объект службы балансов
	*/
	BalanceService struct {
		db repository.SQLStorageInterface
	}

	/*
		Интерфейс оОъекта службы балансов
	*/
	BalanceServiceInterface interface {
		Store(ctx context.Context) error
	}
)

/*
Конструктор службы балансов
*/
func NewBalanceService(db repository.SQLStorageInterface) OrderServiceInterface {
	return &OrderService{
		db: db,
	}
}

/*
Сохраняет баланс
*/
func (s *BalanceService) Store(_ context.Context) error {
	_ = s.db
	return nil
}
