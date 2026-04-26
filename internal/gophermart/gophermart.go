package gophermart

import (
	"context"
	"database/sql"
	"net/http"

	handler "github.com/KalessinD/gophermart/internal/handlers"
	repository "github.com/KalessinD/gophermart/internal/repositories/postgresql"
	service "github.com/KalessinD/gophermart/internal/services"

	"github.com/KalessinD/gophermart/internal/config"
	mw "github.com/KalessinD/gophermart/internal/middleware"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func PsqlConnect(ctx context.Context, dsn string, log *zap.Logger) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Error("Can't connect to psql server", zap.Error(err))
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		log.Error("Can't ping the psql server", zap.Error(err))
		return nil, err
	}

	go func() {
		<-ctx.Done()
		log.Info("Closing database connection due to context cancellation")
		db.Close()
	}()

	log.Info("Successfully connected to the PostgreSQL database")

	return db, nil
}

func GetBaseRouter(cfg *config.GophermartConfig, log *zap.Logger) *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(mw.LoggerMiddleware(log))
	router.Use(middleware.Timeout(cfg.ProcessingTimeout))

	return router
}

func NewRouter(cfg *config.GophermartConfig, log *zap.Logger, pgdb *sql.DB) (http.Handler, error) {
	router := GetBaseRouter(cfg, log)

	commonUserHandler := handler.NewCommonHandler(
		service.NewCommonAction(
			repository.NewSQLStorage(pgdb),
		),
		cfg.EncryptionKey,
	)

	// свободный доступ
	router.Route("/api/user/", func(r chi.Router) {
		r.Post("/login", commonUserHandler.Login)
		r.Post("/register", commonUserHandler.Register)
	})

	restrictedUserHandler := handler.NewRestrictedHandler(
		service.NewCommonAction(
			repository.NewSQLStorage(pgdb),
		),
	)

	// JWT Auth
	router.Group(func(r chi.Router) {
		r.Use(mw.AuthMiddleware)

		r.Post("/orders", restrictedUserHandler.AddOrder)
		r.Get("/orders", restrictedUserHandler.ListOrders)
		r.Get("/balance", restrictedUserHandler.GetLoyalityBalance)
		r.Post("/balance/withdraw", restrictedUserHandler.WithdrawBalance)
		r.Get("/withdrawals", restrictedUserHandler.ListWithdrawals)
	})

	return router, nil
}
