package models_test

import (
	"strings"
	"testing"
	"time"

	"github.com/KalessinD/gophermart/internal/models"
)

func TestNewOrder(t *testing.T) {
	id := "123456789"
	userID := "user_1"
	status := models.OrderNewStatus

	order, err := models.NewOrder(id, userID, status)
	if err != nil {
		t.Fatalf("NewOrder() returned unexpected error: %v", err)
	}
	if order == nil {
		t.Fatal("NewOrder() returned nil order")
	}
	if order.ID != id {
		t.Errorf("Expected ID %s, got %s", id, order.ID)
	}
	if order.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, order.UserID)
	}
	if order.Status != status {
		t.Errorf("Expected Status %s, got %s", status, order.Status)
	}
}

func TestOrder_Validate(t *testing.T) {
	tests := []struct {
		name    string
		order   models.Order
		wantErr bool
		errMsg  string // Подстрока, ожидаемая в ошибке
	}{
		{
			name:    "valid order NEW",
			order:   models.Order{ID: "1", UserID: "u1", Status: models.OrderNewStatus},
			wantErr: false,
		},
		{
			name:    "valid order PROCESSED with accrual",
			order:   models.Order{ID: "1", UserID: "u1", Status: models.OrderProcessedStatus, Accrual: 500},
			wantErr: false,
		},
		{
			name:    "empty UserID",
			order:   models.Order{ID: "1", UserID: "", Status: models.OrderNewStatus},
			wantErr: true,
			errMsg:  "UserID can't be empty",
		},
		{
			name:    "empty ID",
			order:   models.Order{ID: "", UserID: "u1", Status: models.OrderNewStatus},
			wantErr: true,
			errMsg:  "ID can't be equal to zero",
		},
		{
			name:    "invalid status",
			order:   models.Order{ID: "1", UserID: "u1", Status: "UNKNOWN"},
			wantErr: true,
			errMsg:  "wrong status value",
		},
		{
			name:    "negative accrual",
			order:   models.Order{ID: "1", UserID: "u1", Status: models.OrderNewStatus, Accrual: -10},
			wantErr: true,
			errMsg:  "accrual must be a positive or zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.order.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error message = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestOrder_ToJSON(t *testing.T) {
	// Фиксируем время для предсказуемости теста
	now := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		order    models.Order
		wantJSON string // Ожидаемая подстрока или полный JSON
	}{
		{
			name: "accrual is 0 (should be omitted)",
			order: models.Order{
				ID:         "123",
				UserID:     "u1", // Должен быть скрыт
				Status:     models.OrderNewStatus,
				Accrual:    0, // Должен быть скрыт
				UploadedAt: now,
			},
			wantJSON: `{"number":"123","status":"NEW","uploaded":"2023-10-01T12:00:00Z"}`,
		},
		{
			name: "accrual is positive (should be present)",
			order: models.Order{
				ID:         "456",
				UserID:     "u1",
				Status:     models.OrderProcessedStatus,
				Accrual:    500,
				UploadedAt: now,
			},
			wantJSON: `{"number":"456","status":"PROCESSED","accrual":500,"uploaded":"2023-10-01T12:00:00Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.order.ToJSON()
			if err != nil {
				t.Fatalf("ToJSON() returned unexpected error: %v", err)
			}

			gotString := string(data)
			if gotString != tt.wantJSON {
				t.Errorf("ToJSON() = %s, want %s", gotString, tt.wantJSON)
			}

			// убеждаемся, что UserID нет в JSON
			if strings.Contains(gotString, "UserID") {
				t.Error("ToJSON() result should not contain UserID")
			}
			if strings.Contains(gotString, "UpdatedAt") {
				t.Error("ToJSON() result should not contain UpdatedAt")
			}
		})
	}
}
