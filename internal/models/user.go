package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	MinPasswordLength = 8
	MinLoginLength    = 4
)

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrWrongPassword       = errors.New("wrong password")
	ErrUserExists          = errors.New("user exists")
	ErrWrongLoginLength    = fmt.Errorf("login length should be at least %d characters", MinLoginLength)
	ErrWrongPasswordLength = fmt.Errorf("password length should be at least %d characters", MinPasswordLength)
)

type (
	/*
		Объект пользователя системы
	*/
	User struct {
		ID        string    `json:"id"`
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
		Validate() error
	}
)

/*
Конструктор объекта пользователя.
*/
func NewUser(login, password, hash string, version int) *User {
	return &User{Login: login, Password: password, Hash: hash, Version: version}
}

/*
Валидация длины логина и пароля
*/
func (u *User) Validate() error {
	if len(u.Login) < MinLoginLength {
		return ErrWrongLoginLength
	} else if len(u.Password) < MinPasswordLength {
		return ErrWrongPasswordLength
	}
	return nil
}

/*
Сериализует объект пользователя в строку JSON
Может вернуть ошибку, коли та случится.
*/
func (u *User) ToJSON() ([]byte, error) {
	return json.Marshal(u)
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
