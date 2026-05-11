package fixtures

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	AccrualPort = "7091"
)

// AccrualSuite управляет жизненным циклом контейнера accrual
type AccrualSuite struct {
	suite.Suite
	AccrualContainer testcontainers.Container
	AccrualURL       string
	Ctx              context.Context
}

// SetupSuite запускает контейнер
func (s *AccrualSuite) SetupSuite() {
	s.Ctx = context.Background()

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    ".",
			Dockerfile: "docker/accrual/Dockerfile", // Указываем путь к Dockerfile относительно корня проекта
		},
		ExposedPorts: []string{"8080/tcp"},
		WaitingFor:   wait.ForLog("starting server").WithStartupTimeout(30 * time.Second),
		Env: map[string]string{
			"RUN_ADDRESS": "0.0.0.0:" + AccrualPort,
		},
	}

	container, err := testcontainers.GenericContainer(s.Ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(s.T(), err, "Failed to start accrual container")

	s.AccrualContainer = container

	host, err := container.Host(s.Ctx)
	require.NoError(s.T(), err)

	port, err := container.MappedPort(s.Ctx, AccrualPort)
	require.NoError(s.T(), err)

	s.AccrualURL = fmt.Sprintf("http://%s:%s", host, port.Port())
}

// TearDownSuite останавливает контейнер
func (s *AccrualSuite) TearDownSuite() {
	if s.AccrualContainer != nil {
		_ = s.AccrualContainer.Terminate(s.Ctx)
	}
}
