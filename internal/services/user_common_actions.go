package services

import (
	"context"

	"github.com/KalessinD/gophermart/internal/models"
	repository "github.com/KalessinD/gophermart/internal/repositories"

	"golang.org/x/crypto/bcrypt"
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
	if err := userFromRequest.Validate(); err != nil {
		return err
	}

	if err := s.fillUserIfRequired(userFromRequest); err != nil {
		return err
	}

	userFromDb, err := s.db.GetUser(ctx, userFromRequest.Login)
	switch {
	case err != nil:
		return err
	case userFromDb == nil:
		return models.ErrUserNotFound
	case userFromDb.Hash != userFromRequest.Hash:
		return models.ErrWrongPassword
	}

	return nil
}

/*
Выполняет регистрацию новоого пользователя в системе.
Может вернуть ошибку, если таковая произошла
*/
func (s *CommonAction) Register(ctx context.Context, user *models.User) error {
	if err := user.Validate(); err != nil {
		return err
	}
	if err := s.fillUserIfRequired(user); err != nil {
		return err
	}
	return s.db.AddUser(ctx, user)
}

func (s *CommonAction) fillUserIfRequired(user *models.User) error {
	// нечего хешировать, а ничего более тут пока дозаполнять и не надо
	if user.Password == "" || user.Hash == "" {
		return nil
	}
	hash, err := s.hashPassword(user.Password)
	if err == nil {
		user.Hash = hash
	}
	return err
}

func (s *CommonAction) hashPassword(password string) (string, error) {
	// Cost 10 — это баланс между безопасностью и скоростью.
	// Чем выше число, тем медленнее считается хеш, тем сложнее подобрать пароль.
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}
