package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	mw "github.com/KalessinD/gophermart/internal/middleware"
)

func TestMiddleware_LogsRequest(t *testing.T) {
	// создаём observer для перехвата логов
	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	// тестовый handler
	nextHandlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextHandlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// оборачиваем middleware
	middleware := mw.Middleware(logger)
	handler := middleware(next)

	// создаём тестовый запрос
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// вызываем
	handler.ServeHTTP(rec, req)

	if !nextHandlerCalled {
		t.Fatal("next handler was not called")
	}

	logs := recorded.All()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}

	entry := logs[0]

	if entry.Message != "request completed" {
		t.Fatalf("unexpected log message: %s", entry.Message)
	}

	fields := entry.ContextMap()

	if fields["method"] != http.MethodGet {
		t.Errorf("expected method %s, got %v", http.MethodGet, fields["method"])
	}

	if fields["path"] != "/test" {
		t.Errorf("expected path /test, got %v", fields["path"])
	}

	if fields["remote_addr"] == "" {
		t.Error("expected remote_addr to be set")
	}

	if fields["duration"] == nil {
		t.Error("expected duration field")
	}
}
