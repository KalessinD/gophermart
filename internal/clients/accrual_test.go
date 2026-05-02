package clients_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/KalessinD/gophermart/internal/clients"
	mw "github.com/KalessinD/gophermart/internal/middleware"
	"github.com/KalessinD/gophermart/internal/models"
	"github.com/go-chi/chi/middleware"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// MockTransport реализует http.RoundTripper для перехвата HTTP-запросов.
type MockTransport struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func getTestContext(t *testing.T) context.Context {
	t.Helper()

	logger := zaptest.NewLogger(t)
	ctx := t.Context()
	ctx = context.WithValue(ctx, mw.LoggerKey, logger)

	reqID := "test-request-id-123"
	ctx = context.WithValue(ctx, middleware.RequestIDKey, reqID)

	return ctx
}

func TestAccrualClient_GetOrderAccrual(t *testing.T) {
	validOrderID := "12345678903"
	validResponse := models.AccrualResponse{
		Order:   validOrderID,
		Status:  "PROCESSED",
		Accrual: "500",
	}
	validBody, _ := json.Marshal(validResponse)

	tests := []struct {
		name       string
		orderID    string
		mockStatus int
		mockBody   []byte
		mockError  error
		wantErr    error
		wantResp   *models.AccrualResponse
	}{
		{
			name:       "Success - Status OK",
			orderID:    validOrderID,
			mockStatus: http.StatusOK,
			mockBody:   validBody,
			wantResp:   &validResponse,
			wantErr:    nil,
		},
		{
			name:       "Error - Order Not Found (204)",
			orderID:    "999",
			mockStatus: http.StatusNoContent,
			mockBody:   nil,
			wantErr:    clients.ErrOrderNotFound,
			wantResp:   nil,
		},
		{
			name:       "Error - Service Is Busy (429)",
			orderID:    validOrderID,
			mockStatus: http.StatusTooManyRequests,
			mockBody:   []byte("No more than N requests"),
			wantErr:    clients.ErrServiceIsBusy,
			wantResp:   nil,
		},
		{
			name:       "Error - Internal Server Error (500)",
			orderID:    validOrderID,
			mockStatus: http.StatusInternalServerError,
			mockBody:   nil,
			wantErr:    errors.New("unexpected status code: 500"), // Ожидаем ошибку парсинга или статус кода
			wantResp:   nil,
		},
		{
			name:      "Error - Network Error (Retries exceeded)",
			orderID:   validOrderID,
			mockError: errors.New("connection refused"),
			wantErr:   errors.New("request failed"), // Ожидаем ошибку обертку
			wantResp:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := getTestContext(t)

			// Создаем мок транспорта
			transport := &MockTransport{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					// Проверяем метод и заголовки
					require.Equal(t, http.MethodGet, req.Method)
					require.NotEmpty(t, req.Header.Get("X-Request-ID"))
					require.Equal(t, "application/json", req.Header.Get("Accept"))

					if tt.mockError != nil {
						return nil, tt.mockError
					}

					return &http.Response{
						StatusCode: tt.mockStatus,
						Body:       io.NopCloser(bytes.NewReader(tt.mockBody)),
						Header:     make(http.Header),
					}, nil
				},
			}

			httpClient := &http.Client{
				Transport: transport,
			}

			client := &clients.AccrualClient{
				Base: httpClient,
			}

			resp, err := client.GetOrderAccrual(ctx, tt.orderID)

			if tt.wantErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr.Error())
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tt.wantResp.Order, resp.Order)
				require.Equal(t, tt.wantResp.Status, resp.Status)
				require.Equal(t, tt.wantResp.Accrual, resp.Accrual)
			}
		})
	}
}
