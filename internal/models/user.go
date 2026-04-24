package models

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrUserNotFound = errors.New("user not found")

type (
	/*
		Объект пользователя системы
	*/
	User struct {
		Login     string    `json:"login"`
		Password  string    `json:"-"`
		Hash      string    `json:"-"`
		Salt      string    `json:"-"`
		CreatedAt time.Time `json:"hash,omitempty"`
	}

	/*
		Интерфейс объекта пользователя системы
	*/
	UserInterface interface {
		ToJSON() ([]byte, error)
	}
)

/*
Конструктор объекта пользователя.
*/
func NewUser(login, password, hash, salt string) *User {
	return &User{Login: login, Password: password, Hash: hash, Salt: salt}
}

/*
Сериализует объект пользователя в строку JSON
Может вернуть ошибку, коли та случится.
*/
func (m *User) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

/*
Десериализует строку JSON в объект пользователя.
Может вернуть ошибку, коли та случится.
*/
func FromJSON(str []byte) (*User, error) {
	metric := &User{}
	err := json.Unmarshal(str, metric)
	if err != nil {
		return nil, err
	}
	return metric, nil
}
