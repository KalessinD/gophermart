package gophermart

import (
	"context"
	"database/sql"
	"net/http"

	// "github.com/KalessinD/gophermart/internal/config"
	// "github.com/KalessinD/gophermart/internal/handlers"
	// "github.com/KalessinD/gophermart/internal/repositories"
	// "github.com/KalessinD/gophermart/internal/services"

	"github.com/KalessinD/gophermart/internal/config"
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

	// router.Use(middleware.Logger) // slow
	// router.Use(middleware.RequestID)

	// router.Use(mw.RequestIDMiddleware)
	// router.Use(middleware.Recoverer)
	// router.Use(mw.ChiParamsMiddleware)
	// router.Use(mw.Middleware(log))
	// router.Use(mw.CompressionMiddleware)
	// router.Use(mw.CheckHashSum(cfg.EncryptionKey))
	// router.Use(middleware.Timeout(cfg.ProcessingTimeout))

	return router
}

func NewRouter(ctx context.Context, cfg *config.GophermartConfig, log *zap.Logger) (http.Handler, error) {
	router := GetBaseRouter(cfg, log)

	return router, nil
}
