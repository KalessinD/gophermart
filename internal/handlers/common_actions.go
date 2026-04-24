package handler

import (
	"net/http"

	// "github.com/KalessinD/gophermart/internal/models"
	middleware "github.com/KalessinD/gophermart/internal/middleware"
	service "github.com/KalessinD/gophermart/internal/services"
)

const HeaderContentTypeJSON = "application/json"

type CommonHandler struct {
	metricsService service.UserCommonActions
}

func NewCommonHandler(metricService service.UserCommonActions) *CommonHandler {
	return &CommonHandler{
		metricsService: metricService,
	}
}

func (h *CommonHandler) Login(w http.ResponseWriter, r *http.Request) {
	log := middleware.GetLogger(r.Context())

	if r.Method != http.MethodPost {
		log.Debug("Not allowed")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *CommonHandler) Register(w http.ResponseWriter, r *http.Request) {
	log := middleware.GetLogger(r.Context())

	if r.Method != http.MethodPost {
		log.Debug("Not allowed")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
}
