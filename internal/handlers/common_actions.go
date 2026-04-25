package handlers

import (
	"io"
	"net/http"

	// "github.com/KalessinD/gophermart/internal/models"
	middleware "github.com/KalessinD/gophermart/internal/middleware"
	model "github.com/KalessinD/gophermart/internal/models"
	service "github.com/KalessinD/gophermart/internal/services"
)

const HeaderContentTypeJSON = "application/json"

type CommonHandler struct {
	commonService service.UserCommonActions
}

func NewCommonHandler(commonService service.UserCommonActions) *CommonHandler {
	return &CommonHandler{
		commonService: commonService,
	}
}

func (h *CommonHandler) Login(w http.ResponseWriter, r *http.Request) {
	log := middleware.GetLogger(r.Context())

	if r.Method != http.MethodPost {
		log.Debug("Not allowed")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Debug("Bad request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Sugar().Debugf("Bad request: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := model.FromJSON(body)
	if err != nil {
		log.Sugar().Debugf("Can't parse JSON: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = h.commonService.Login(r.Context(), user)

	_ = err

	w.WriteHeader(http.StatusOK)
}

func (h *CommonHandler) Register(w http.ResponseWriter, r *http.Request) {
	log := middleware.GetLogger(r.Context())

	if r.Method != http.MethodPost {
		log.Debug("Not allowed")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Debug("Bad request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Sugar().Debugf("Bad request: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := model.FromJSON(body)
	if err != nil {
		log.Sugar().Debugf("Can't parse JSON: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = h.commonService.Register(r.Context(), user)

	_ = err

	w.WriteHeader(http.StatusOK)
}
