package servives

import (
	"context"
	"errors"

	"github.com/KalessinD/gophermart/internal/models"
	repository "github.com/KalessinD/gophermart/internal/repositories"
)

type (
	/*
		Объект службы для действий пользователя, нетребующих авторизации
	*/
	CommonAction struct {
		db repository.SQLStorageInterface
	}

	/*
		Интерфейс оОбъекта службы для действий пользователя, нетребующих авторизации
	*/
	UserCommonActions interface {
		Login(ctx context.Context, user *models.User) error
		Register(ctx context.Context, user *models.User) error
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
func (s *CommonAction) Login(ctx context.Context, userFromRequest *models.User) error {
	if err := s.validate(userFromRequest); err != nil {
		return err
	}
	userFromDb, err := s.db.GetUser(ctx, userFromRequest.Login)
	_ = userFromDb
	return err
}

/*
Выполняет регистрацию новоого пользователя в системе.
Может вернуть ошибку, если таковая произошла
*/
func (s *CommonAction) Register(ctx context.Context, user *models.User) error {
	if err := s.validate(user); err != nil {
		return err
	}
	return s.db.AddUser(ctx, user)
}

func (s *CommonAction) validate(user *models.User) error {
	if user.Login == "" {
		return errors.New("wrong login format")
	} else if user.Password == "" {
		return errors.New("wrong password format")
	}
	return nil
}
