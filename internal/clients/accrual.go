package clients

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	mw "github.com/KalessinD/gophermart/internal/middleware"
	"github.com/KalessinD/gophermart/internal/models"
	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
)

const (
	RetryingAttempts  = 3
	RetryingDelay     = 100 * time.Microsecond
	RetryingDelayStep = 200 * time.Microsecond

	OrderBaseURL = "api/orders/"
)

var (
	ErrServiceIsBusy = errors.New("accrual service responds about too many requests")
)

type (
	AccrualClient struct {
		base *http.Client
	}

	AccrualClienttInterface interface {
		GetOrderAccrual(ctx context.Context, orderID string) (*models.AccrualResponse, error)
	}
)

// Конструктор http-клиента для запросов в систему Accrual
func NewAccrualClient() AccrualClienttInterface {
	return &AccrualClient{
		base: &http.Client{},
	}
}

// Выполняет запрос с попытками реконнекта
func (c *AccrualClient) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	attempts := RetryingAttempts
	delay := RetryingDelay

	var err error

	for attempts > 0 {
		// nolint:gosec
		response, err := c.base.Do(req)

		if err == nil {
			return response, nil
		}

		if attempts == 1 {
			return nil, err
		}

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		attempts--
		delay += RetryingDelayStep
	}

	return nil, err
}

/*

Для взаимодействия с системой доступен один хендлер:
GET /api/orders/{number} — получение информации о расчёте начислений баллов лояльности.

Формат запроса:

````
GET /api/orders/{number} HTTP/1.1
Content-Length: 0
```
`
Возможные коды ответа:
- 200 — успешная обработка запроса.
  Формат ответа:
```
  200 OK HTTP/1.1
  Content-Type: application/json
	...

  {
      "order": "<number>",
      "status": "PROCESSED",
      "accrual": 500
  }
```
  Поля объекта ответа:
	- order — номер заказа;
	- status — статус расчёта начисления:
		- REGISTERED — заказ зарегистрирован, но вознаграждение не рассчитано;
		- INVALID — заказ не принят к расчёту, и вознаграждение не будет начислено;
		- PROCESSING — расчёт начисления в процессе;
		- PROCESSED — расчёт начисления окончен;
	- accrual — рассчитанные баллы к начислению, при отсутствии начисления — поле отсутствует в ответе.
- 204 — заказ не зарегистрирован в системе расчёта.
- 429 — превышено количество запросов к сервису.
	  Формат ответа:
```
  429 Too Many Requests HTTP/1.1
  Content-Type: text/plain
  Retry-After: 60
`
  No more than N requests per minute allowed
```
- 500 — внутренняя ошибка сервера.

Заказ может быть взят в расчёт в любой момент после его совершения.
Время выполнения расчёта системой не регламентировано.
Статусы INVALID и PROCESSED являются окончательными.
Общее количество запросов информации о начислении не ограничено.

*/

func (c *AccrualClient) GetOrderAccrual(ctx context.Context, orderID string) (*models.AccrualResponse, error) {
	log := mw.GetLogger(ctx)
	requestID := middleware.GetReqID(ctx)
	url := OrderBaseURL + orderID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("X-Request-ID", requestID)

	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("can't send a request: %w", err)
	}

	log.Info(
		"Request was sent",
		zap.String("request_id", requestID),
		zap.String("method", req.Method),
		zap.String("url", url),
		zap.String("response-status", resp.Status),
	)

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrServiceIsBusy
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read a response body: %w", err)
	}

	accrualResp, err := models.FromJSON[models.AccrualResponse](body)
	if err != nil {
		return nil, fmt.Errorf("can't parse a response body: %w", err)
	}

	log.Info(
		"Request was sent",
		zap.String("request_id", requestID),
		zap.String("method", req.Method),
		zap.String("url", url),
		zap.String("response-status", resp.Status),
	)

	return accrualResp, nil
}
