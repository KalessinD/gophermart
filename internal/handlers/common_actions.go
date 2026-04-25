package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	middleware "github.com/KalessinD/gophermart/internal/middleware"
	model "github.com/KalessinD/gophermart/internal/models"
	service "github.com/KalessinD/gophermart/internal/services"
)

type (
	CommonHandler struct {
		commonService service.UserCommonActions
	}

	CommonHandlerInterface interface {
		Login(w http.ResponseWriter, r *http.Request)
		Register(w http.ResponseWriter, r *http.Request)
	}
)

/*
Конструктор для хендлеров работающих без автоиизации
*/
func NewCommonHandler(commonService service.UserCommonActions) CommonHandlerInterface {
	return &CommonHandler{
		commonService: commonService,
	}
}

/*
Аутентификация пользователя

Формат запроса:
```
POST /api/user/login HTTP/1.1
Content-Type: application/json
...

	{
	    "login": "<login>",
	    "password": "<password>"
	}

```

Возможные коды ответа:
- 200 — пользователь успешно аутентифицирован;
- 400 — неверный формат запроса;
- 401 — неверная пара логин/пароль;
- 500 — внутренняя ошибка сервера.
*/
func (h *CommonHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := middleware.GetLogger(ctx)
	user, err := h.commonChecks(w, r)
	if err != nil {
		log.Sugar().Debugf("login failed: %s", err)
		return
	}

	if err = h.commonService.Login(ctx, user); err != nil {
		log.Sugar().Debugf("login failed: %s", err)
		h.setResponseStatusByError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

/*
Регистрация пользователя

Формат запроса:
```
POST /api/user/register HTTP/1.1
Content-Type: application/json
...

	{
	    "login": "<login>",
	    "password": "<password>"
	}

```
Возможные коды ответа:
- 200 — пользователь успешно зарегистрирован и аутентифицирован;
- 400 — неверный формат запроса;
- 409 — логин уже занят;
- 500 — внутренняя ошибка сервера.
*/
func (h *CommonHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := middleware.GetLogger(ctx)
	user, err := h.commonChecks(w, r)
	if err != nil {
		log.Sugar().Debugf("user registration failed: %s", err)
		return
	}

	if err = h.commonService.Register(ctx, user); err != nil {
		log.Sugar().Debugf("user registration failed: %s", err)
		h.setResponseStatusByError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *CommonHandler) commonChecks(w http.ResponseWriter, r *http.Request) (user *model.User, err error) {
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		err = fmt.Errorf("bad request: %s", "wrong content-type")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("bad request: %w", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err = model.FromJSON(body)
	if err != nil {
		err = fmt.Errorf("can't parse request body: %w", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	return
}

func (h *CommonHandler) setResponseStatusByError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, model.ErrWrongLoginLength):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Is(err, model.ErrWrongPasswordLength):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Is(err, model.ErrUserNotFound):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Is(err, model.ErrWrongPassword):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Is(err, model.ErrUserExists):
		w.WriteHeader(http.StatusConflict)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}
