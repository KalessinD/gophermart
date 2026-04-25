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
		Version   int       `json:"version"`
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
func NewUser(login, password, hash string, version int) *User {
	return &User{Login: login, Password: password, Hash: hash, Version: version}
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
func FromJSON(str []byte) (user *User, err error) {
	err = json.Unmarshal(str, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}
