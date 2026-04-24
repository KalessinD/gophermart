package servives

import (
	"errors"

	"github.com/KalessinD/gophermart/internal/models"
	repository "github.com/KalessinD/gophermart/internal/repositories"
)

type (
	CommonAction struct {
		db repository.SQLStorageInterface
	}

	UserCommonActions interface {
		Login(user *models.User) error
		Register(user *models.User) error
	}
)

/*
Конструктор службы для операций нетребующих авторизации пользователя.
*/
func NewCommonAction(db repository.SQLStorageInterface) UserCommonActions {
	return &CommonAction{
		db: db,
	}
}

/*
Выполняет вход в систему.
Может вернуть ошибку, если таковая произошла
*/
func (s *CommonAction) Login(user *models.User) error {
	return s.validate(user)
}

/*
Выполняет регистрацию новоого пользователя в системе.
Может вернуть ошибку, если таковая произошла
*/
func (s *CommonAction) Register(user *models.User) error {
	return s.validate(user)
}

func (s *CommonAction) validate(user *models.User) error {
	if user.Login == "" {
		return errors.New("wrong login format")
	} else if user.Password == "" {
		return errors.New("wrong password format")
	}
	return nil
}
