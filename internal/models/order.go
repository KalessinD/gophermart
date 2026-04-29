package models

import (
	"encoding/json"
	"errors"
	"time"
)

const (
	OrderNewStatus       = "NEW"
	OrderInProcessStatus = "PROCESSING"
	OrderInvalidStatus   = "INVALID"
	OrderProcessedStatus = "PROCESSED"
)

var (
	ErrOrderNotFound           = errors.New("order not found")
	ErrOrderExists             = errors.New("order exists")
	ErrOrderBelongsToOtherUser = errors.New("order belongs to other user")
	ErrOrderWrongFormat        = errors.New("wrong order format")

	validStatuses = map[string]bool{
		OrderNewStatus:       true,
		OrderInProcessStatus: true,
		OrderInvalidStatus:   true,
		OrderProcessedStatus: true,
	}
)

type (
	/*
		Объект пользователя системы
	*/
	Order struct {
		ID         string    `json:"number"`
		UserID     string    `json:"-"`
		Status     string    `json:"status"`
		Accrual    int       `json:"accrual,omitempty"`
		UploadedAt time.Time `json:"uploaded"`
		UpdatedAt  time.Time `json:"-"`
	}

	/*
		Интерфейс объекта пользователя системы
	*/
	OrderInterface interface {
		ToJSON() ([]byte, error)
		Validate() error
	}

	OrdersList []*Order
)

/*
Конструктор объекта заказа.
*/
func NewOrder(id, userID, status string) (*Order, error) {
	return &Order{ID: id, UserID: userID, Status: status}, nil
}

/*
Валидация заказа
*/
func (m *Order) Validate() error {
	switch {
	case m.UserID == "":
		return errors.New("UserID can't be empty")
	case m.ID == "":
		return errors.New("ID can't be equal to zero")
	case !validStatuses[m.Status]:
		return errors.New("wrong status value")
	case m.Accrual < 0:
		return errors.New("accrual must be a positive or zero")
	}
	return nil
}

/*
Сериализует объект пользователя в строку JSON
Может вернуть ошибку, коли та случится.
*/
func (m *Order) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}
