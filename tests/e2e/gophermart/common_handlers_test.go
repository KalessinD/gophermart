//go:build e2e

package gophermart_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KalessinD/gophermart/internal/config"
	"github.com/KalessinD/gophermart/internal/gophermart"
	"github.com/KalessinD/gophermart/internal/models"
	"github.com/golang-migrate/migrate/v4"
	mpg "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

type HandlerE2ETestSuite struct {
	suite.Suite
	container *postgres.PostgresContainer
	db        *sql.DB
	server    *httptest.Server
	log       *zap.Logger
	ctx       context.Context
}

// SetupSuite запускается один раз перед всеми тестами в Suite.
func (s *HandlerE2ETestSuite) SetupSuite() {
	s.log, _ = zap.NewDevelopment()
	s.ctx = context.Background()

	var err error

	s.container, err = postgres.Run(s.ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("gophermart_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(10*time.Second),
		),
	)
	require.NoError(s.T(), err, "Failed to start container")

	// Получаем строку подключения
	connStr, err := s.container.ConnectionString(s.ctx, "sslmode=disable")
	require.NoError(s.T(), err, "Failed to get connection string")

	s.db, err = sql.Open("pgx", connStr)
	require.NoError(s.T(), err, "Failed to connect to DB")

	// Применяем миграции
	s.applyMigrations("file://../../../migrations")

	// Создаем конфиг
	cfg := &config.GophermartConfig{
		ListenAddr:               ":9081",
		ProcessingTimeout:        60 * time.Second,
		ReadTimeout:              5 * time.Second,
		ReadHeaderTimeout:        5 * time.Second,
		WriteTimeout:             10 * time.Second,
		IdleTimeout:              30 * time.Second,
		GracefullShutdownTimeout: 5 * time.Second,
		PsqlDSN:                  connStr,
		EncryptionKey:            "test-secret-key-for-e2e",
		AccrualAddress:           "",
		QueueBufSize:             4,
		QueueWorkers:             10,
		AccrualClientTImeout:     3 * time.Second,
		WorkerPoolChanBuffer:     32,
		DumperStoragePath:        filepath.Join(s.T().TempDir(), "server.dump"),
	}

	// Инициализируем роутер (Приложение)
	router, err := gophermart.NewRouter(s.ctx, cfg, s.log, s.db)
	require.NoError(s.T(), err, "Failed to create router")

	s.server = httptest.NewServer(router)
}

func (s *HandlerE2ETestSuite) applyMigrations(sourceURL string) {
	driver, err := mpg.WithInstance(s.db, &mpg.Config{})
	require.NoError(s.T(), err, "Failed to create migrate driver")

	m, err := migrate.NewWithDatabaseInstance(sourceURL, "gophermart_test", driver)
	require.NoError(s.T(), err, "Failed to create migrate instance")

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		require.NoError(s.T(), err, "Failed to apply migrations")
	}
}

func (s *HandlerE2ETestSuite) TearDownSuite() {
	if s.server != nil {
		s.server.Close()
	}
	if s.db != nil {
		s.db.Close()
	}
	if s.container != nil {
		if err := s.container.Terminate(s.ctx); err != nil {
			s.log.Sugar().Errorf("Failed to terminate container: %v", err)
		}
	}
}

func (s *HandlerE2ETestSuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, "TRUNCATE TABLE gophermart.users CASCADE")
	require.NoError(s.T(), err, "Failed to truncate users")
}

func TestHandlerE2ETestSuite(t *testing.T) {
	suite.Run(t, new(HandlerE2ETestSuite))
}

// --- Тесты ---

func (s *HandlerE2ETestSuite) TestRegister_Success() {
	// Подготовка данных
	user := models.User{Login: "testuser", Password: "password123"}
	body, _ := json.Marshal(user)

	// Выполнение запроса
	resp, err := http.Post(s.server.URL+"/api/user/register", "application/json", strings.NewReader(string(body)))
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	// Проверки
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	// Проверяем наличие куки
	cookies := resp.Cookies()
	require.NotEmpty(s.T(), cookies, "Auth cookie should be set")

	var found bool
	for _, c := range cookies {
		if c.Name == "token" {
			assert.NotEmpty(s.T(), c.Value, "Token should not be empty")
			found = true
		}
	}
	assert.True(s.T(), found, "Token cookie not found")
}

func (s *HandlerE2ETestSuite) TestRegister_Conflict() {
	// Сначала регистрируем
	user := models.User{Login: "conflict_user", Password: "password123Е"}
	body, _ := json.Marshal(user)
	_, err := http.Post(s.server.URL+"/api/user/register", "application/json", strings.NewReader(string(body)))
	require.NoError(s.T(), err)

	// Пытаемся зарегистрировать снова с тем же логином
	resp, err := http.Post(s.server.URL+"/api/user/register", "application/json", strings.NewReader(string(body)))
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusConflict, resp.StatusCode)
}

func (s *HandlerE2ETestSuite) TestLogin_Success() {
	// Сначала регистрируем пользователя
	user := models.User{Login: "loginuser", Password: "password"}
	regBody, _ := json.Marshal(user)
	_, err := http.Post(s.server.URL+"/api/user/register", "application/json", strings.NewReader(string(regBody)))
	require.NoError(s.T(), err)

	// Теперь логинимся
	loginBody, _ := json.Marshal(user)
	resp, err := http.Post(s.server.URL+"/api/user/login", "application/json", strings.NewReader(string(loginBody)))
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	// Проверяем куку
	cookies := resp.Cookies()
	require.NotEmpty(s.T(), cookies)
	var found bool
	for _, c := range cookies {
		if c.Name == "token" {
			assert.NotEmpty(s.T(), c.Value)
			found = true
		}
	}
	assert.True(s.T(), found)
}

func (s *HandlerE2ETestSuite) TestLogin_Unauthorized() {
	// Пытаемся войти под несуществующим пользователем
	user := models.User{Login: "nonexistent", Password: "wrong-password"}
	body, _ := json.Marshal(user)

	resp, err := http.Post(s.server.URL+"/api/user/login", "application/json", strings.NewReader(string(body)))
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (s *HandlerE2ETestSuite) TestRegister_BadRequest() {
	// Отправляем невалидный JSON
	resp, err := http.Post(s.server.URL+"/api/user/register", "application/json", strings.NewReader("{bad json}"))
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode)
}
