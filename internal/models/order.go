package models

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

var (
	ErrOrderNotFound           = errors.New("order not found")
	ErrOrderExists             = errors.New("order exists")
	ErrOrderBelongsToOtherUser = errors.New("order belongs to other user")
)

type (
	/*
		Объект пользователя системы
	*/
	Order struct {
		ID         int64     `json:"id"`
		UserID     string    `json:"user_id"`
		Status     string    `json:"status"`
		Accrual    int       `json:"accrual"`
		UploadedAt time.Time `json:"uploaded,omitempty"`
		UpdatedAt  time.Time `json:"updated,omitempty"`
	}

	/*
		Интерфейс объекта пользователя системы
	*/
	OrderInterface interface {
		ToJSON() ([]byte, error)
		Validate() error
	}
)

/*
Конструктор объекта заказа.
*/
func NewOrder(idStr, userID string) (*Order, error) {
	idInt, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, err
	}
	return &Order{ID: idInt, UserID: userID}, nil
}

/*
Валидация заказа
*/
func (m *Order) Validate() error {
	return nil
}

/*
Сериализует объект пользователя в строку JSON
Может вернуть ошибку, коли та случится.
*/
func (m *Order) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}
