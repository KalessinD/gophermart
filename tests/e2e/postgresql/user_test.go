//go:build e2e

package postgresql_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/KalessinD/gophermart/internal/models"
	"github.com/KalessinD/gophermart/internal/repositories/postgresql"
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
)

type UserE2ETestSuite struct {
	suite.Suite
	container *postgres.PostgresContainer
	db        *sql.DB
	storage   postgresql.SQLStorageInterface
	ctx       context.Context
}

// SetupSuite запускается ОДИН РАЗ перед всеми тестами.
// Здесь мы поднимаем контейнер и накатываем миграции.
func (s *UserE2ETestSuite) SetupSuite() {
	s.ctx = context.Background()

	var err error

	// Запускаем Docker-контейнер с PostgreSQL
	s.container, err = postgres.Run(s.ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("gophermart_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5*time.Second),
		),
	)
	require.NoError(s.T(), err, "Failed to start container")

	// Получаем строку подключения из контейнера
	connStr, err := s.container.ConnectionString(s.ctx, "sslmode=disable")
	require.NoError(s.T(), err, "Failed to get connection string")

	// Подключаемся к БД
	s.db, err = sql.Open("pgx", connStr)
	require.NoError(s.T(), err, "Failed to connect to DB")

	// Проверяем коннект
	err = s.db.PingContext(s.ctx)
	require.NoError(s.T(), err, "Failed to ping DB")

	// путь к миграциям указывается относительно того, откуда запускаются тесты.
	s.applyMigrations("file://../../../migrations")

	s.storage = postgresql.NewSQLStorage(s.db)
}

// applyMigrations применяет схемы к базе
func (s *UserE2ETestSuite) applyMigrations(sourceURL string) {
	driver, err := mpg.WithInstance(s.db, &mpg.Config{})
	require.NoError(s.T(), err, "Failed to create migrate driver")

	m, err := migrate.NewWithDatabaseInstance(sourceURL, "gophermart_test", driver)
	require.NoError(s.T(), err, "Failed to create migrate instance")

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		require.NoError(s.T(), err, "Failed to apply migrations")
	}
}

// TearDownSuite убивает контейнер после всех тестов
func (s *UserE2ETestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
	if s.container != nil {
		if err := s.container.Terminate(s.ctx); err != nil {
			s.T().Logf("Failed to terminate container: %v", err)
		}
	}
}

func (s *UserE2ETestSuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, "TRUNCATE TABLE gophermart.users CASCADE")
	require.NoError(s.T(), err, "Failed to truncate users")
}

func TestUserE2ETestSuite(t *testing.T) {
	suite.Run(t, new(UserE2ETestSuite))
}

// --- Тесты ---

func (s *UserE2ETestSuite) TestAddAndGetUser() {
	user := &models.User{
		Login: "test_user",
		Hash:  "secret_hash",
	}

	// Create
	err := s.storage.AddUser(s.ctx, user)
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), user.ID)

	// Read
	foundUser, err := s.storage.GetUser(s.ctx, "test_user")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), user.ID, foundUser.ID)
	assert.Equal(s.T(), "secret_hash", foundUser.Hash)
}

func (s *UserE2ETestSuite) TestAddUserDuplicate() {
	user := &models.User{Login: "dup_user", Hash: "h1"}
	err := s.storage.AddUser(s.ctx, user)
	require.NoError(s.T(), err)

	// Попытка создать дубликат
	dupUser := &models.User{Login: "dup_user", Hash: "h2"}
	err = s.storage.AddUser(s.ctx, dupUser)

	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, models.ErrUserExists)
}
