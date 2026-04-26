package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/KalessinD/gophermart/internal/common"
	"github.com/KalessinD/gophermart/internal/models"
	"github.com/KalessinD/gophermart/internal/repositories/postgresql/mocks"
	"github.com/KalessinD/gophermart/internal/services"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

// тестирует регистрацию пользователя
func TestCommonAction_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockSQLStorageInterface(ctrl)
	svc := services.NewCommonAction(mockDB)

	t.Run("successful registration", func(t *testing.T) {
		ctx := t.Context()
		password := "password123"
		user := &models.User{
			Login:    "testuser",
			Password: password,
		}

		// Ожидаем, что AddUser будет вызван.
		// Используем Do, чтобы проверить, что пароль был захеширован перед сохранением.
		mockDB.EXPECT().AddUser(ctx, gomock.Any()).DoAndReturn(
			func(ctx context.Context, u *models.User) error {
				if u.Login != "testuser" {
					t.Errorf("expected login testuser, got %s", u.Login)
				}
				if u.Hash == "" {
					t.Error("expected hash to be generated, got empty string")
				}
				if u.Hash == password {
					t.Error("password should be hashed, not plain text")
				}
				// Проверяем, что хеш валиден
				if err := bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(password)); err != nil {
					t.Errorf("generated hash is not valid for password: %v", err)
				}
				return nil
			},
		)

		err := svc.Register(ctx, user)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		ctx := t.Context()
		user := &models.User{
			Login:    "us", // слишком короткий логин
			Password: "password123",
		}

		// База данных не должна вызываться
		err := svc.Register(ctx, user)
		if err == nil {
			t.Error("expected validation error, got nil")
		}
	})
}

// тестирует вход пользователя
func TestCommonAction_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockSQLStorageInterface(ctrl)
	svc := services.NewCommonAction(mockDB)

	password := "password123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate hash for test: %v", err)
	}

	userFromDB := &models.User{
		Login: "testuser",
		Hash:  string(hash),
	}

	t.Run("successful login", func(t *testing.T) {
		ctx := t.Context()
		userReq := &models.User{
			Login:    "testuser",
			Password: password,
		}

		// Мокаем получение пользователя из БД
		mockDB.EXPECT().GetUser(ctx, "testuser").Return(userFromDB, nil)

		err := svc.Login(ctx, userReq)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		ctx := t.Context()
		userReq := &models.User{
			Login:    "nonexistent",
			Password: "password123",
		}

		mockDB.EXPECT().GetUser(ctx, "nonexistent").Return(nil, nil) // GetUser возвращает nil, nil

		err := svc.Login(ctx, userReq)
		if !errors.Is(err, models.ErrUserNotFound) {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		ctx := t.Context()
		userReq := &models.User{
			Login:    "testuser",
			Password: "wrongpassword",
		}

		mockDB.EXPECT().GetUser(ctx, "testuser").Return(userFromDB, nil)

		err := svc.Login(ctx, userReq)
		if !errors.Is(err, models.ErrWrongPassword) {
			t.Errorf("expected ErrWrongPassword, got %v", err)
		}
	})
}

// тестирует генерацию JWT
func TestCommonAction_GenerateToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockSQLStorageInterface(ctrl)
	svc := services.NewCommonAction(mockDB)

	t.Run("token generation", func(t *testing.T) {
		user := &models.User{
			ID:    "123",
			Login: "testuser",
		}
		expireAt := time.Now().Add(time.Hour)
		secret := "test-secret"

		tokenStr, err := svc.GenerateToken(user, secret, expireAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Парсим токен для проверки содержимого
		claims := &common.Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		})

		if err != nil {
			t.Fatalf("failed to parse token: %v", err)
		}

		if !token.Valid {
			t.Error("token is not valid")
		}

		if claims.UserID != "123" {
			t.Errorf("expected userID 123, got %s", claims.UserID)
		}
		if claims.Login != "testuser" {
			t.Errorf("expected login testuser, got %s", claims.Login)
		}
	})
}
