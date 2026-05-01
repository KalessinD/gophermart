package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/KalessinD/gophermart/internal/common"
	"github.com/KalessinD/gophermart/internal/middleware"
	model "github.com/KalessinD/gophermart/internal/models"
	repoMocks "github.com/KalessinD/gophermart/internal/repositories/postgresql/mocks"
	"github.com/KalessinD/gophermart/internal/services"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// --- Вспомогательные функции ---

// getCtxWithClaims создает контекст с фиктивными данными пользователя (Claims)
// Это необходимо, так как Service вызывает middleware.GetClaims(ctx)
func getCtxWithClaims(userID string) context.Context {
	// Создаем структуру Claims, аналогичную той, что используется в middleware
	claims := &common.Claims{
		UserID: userID,
	}
	// Кладем в контекст. Предполагаем, что middleware.ContextClaimsKey экспортируется или доступна.
	// Если ключ приватный, вам нужно будет добавить функцию-хелпер в пакет middleware.
	return context.WithValue(context.Background(), middleware.ClaimsKey, claims)
}

// --- Тесты ---

func TestOrderActions_Store(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Создаем мок репозитория
	mockRepo := repoMocks.NewMockSQLStorageInterface(ctrl)

	// Создаем службу с моком
	service := services.NewOrderActions(mockRepo)

	// Валидный номер заказа по Луну (стандартный тестовый номер)
	validOrderID := "79927398713"
	userID := "user_1"

	ctx := getCtxWithClaims(userID)

	tests := []struct {
		name       string
		orderID    string
		mockExpect func()
		wantErr    error
	}{
		{
			name:    "Success - new order",
			orderID: validOrderID,
			mockExpect: func() {
				// Ожидаем, что заказ успешно добавится в БД
				mockRepo.EXPECT().
					AddOrder(gomock.Any(), gomock.Any()). // Можно проверить конкретный объект заказа через gomock.Eq, если важно точное совпадение
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name:    "Error - invalid Luhn format",
			orderID: "123", // Невалидный номер
			mockExpect: func() {
				// БД не должна вызываться
			},
			wantErr: model.ErrOrderWrongFormat,
		},
		{
			name:    "Error - order belongs to other user",
			orderID: validOrderID,
			mockExpect: func() {
				// 1. Попытка добавить возвращает "уже существует"
				mockRepo.EXPECT().
					AddOrder(gomock.Any(), gomock.Any()).
					Return(model.ErrOrderExists)

				// 2. Служба проверяет, чей это заказ
				mockRepo.EXPECT().
					GetOrder(gomock.Any(), validOrderID, userID).
					Return(nil, model.ErrOrderNotFound) // Для этого юзера не найден -> значит чужой
			},
			wantErr: model.ErrOrderBelongsToOtherUser,
		},
		{
			name:    "Error - order already exists for current user",
			orderID: validOrderID,
			mockExpect: func() {
				// 1. Попытка добавить возвращает "уже существует"
				mockRepo.EXPECT().
					AddOrder(gomock.Any(), gomock.Any()).
					Return(model.ErrOrderExists)

				// 2. Служба проверяет, чей это заказ
				mockRepo.EXPECT().
					GetOrder(gomock.Any(), validOrderID, userID).
					Return(&model.Order{ID: validOrderID, UserID: userID}, nil) // Найден для этого юзера
			},
			wantErr: model.ErrOrderExists,
		},
		{
			name:    "Error - internal DB error on AddOrder",
			orderID: validOrderID,
			mockExpect: func() {
				mockRepo.EXPECT().
					AddOrder(gomock.Any(), gomock.Any()).
					Return(errors.New("connection lost"))
			},
			wantErr: errors.New("connection lost"), // Служба возвращает ошибку как есть
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockExpect()

			err := service.Store(ctx, tt.orderID)

			if tt.wantErr != nil {
				require.Error(t, err)
				// Проверяем текст ошибки или тип
				if errors.Is(tt.wantErr, model.ErrOrderWrongFormat) || errors.Is(tt.wantErr, model.ErrOrderExists) {
					require.ErrorIs(t, err, tt.wantErr)
				} else {
					require.EqualError(t, err, tt.wantErr.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOrderActions_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repoMocks.NewMockSQLStorageInterface(ctrl)
	service := services.NewOrderActions(mockRepo)

	userID := "user_1"
	ctx := getCtxWithClaims(userID)

	tests := []struct {
		name       string
		mockExpect func()
		wantList   model.OrdersList
		wantErr    error
	}{
		{
			name: "Success - list found",
			mockExpect: func() {
				orders := model.OrdersList{
					&model.Order{ID: "1", UserID: userID},
				}
				mockRepo.EXPECT().
					ListOrders(gomock.Any(), userID).
					Return(orders, nil)
			},
			wantList: model.OrdersList{
				&model.Order{ID: "1", UserID: userID},
			},
			wantErr: nil,
		},
		{
			name: "Success - empty list (no orders)",
			mockExpect: func() {
				mockRepo.EXPECT().
					ListOrders(gomock.Any(), userID).
					Return(model.OrdersList{}, nil) // Репозиторий возвращает пустой список
			},
			wantList: model.OrdersList{},
			wantErr:  nil,
		},
		{
			name: "Error - DB error",
			mockExpect: func() {
				mockRepo.EXPECT().
					ListOrders(gomock.Any(), userID).
					Return(nil, errors.New("db error"))
			},
			wantList: nil,
			wantErr:  errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockExpect()

			list, err := service.List(ctx)

			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr.Error())
				require.Nil(t, list)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantList, list)
			}
		})
	}
}
