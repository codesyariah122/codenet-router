package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codesyariah/codenet-router/apps/api/internal/providers"
	"github.com/codesyariah/codenet-router/apps/api/internal/services"
)

func testHandler() http.Handler {
	return New(services.NewRouter(providers.NewMock()), slog.New(slog.NewTextHandler(io.Discard, nil)), "http://127.0.0.1:5173")
}

func TestDashboardResponse(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	recorder := httptest.NewRecorder()
	testHandler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var payload map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["success"] != true {
		t.Fatal("success harus true")
	}
	data := payload["data"].(map[string]any)
	if _, ok := data["system"]; !ok {
		t.Fatal("data system tidak tersedia")
	}
}

func TestInterfaceNotFound(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/interfaces/tidak-ada", nil)
	recorder := httptest.NewRecorder()
	testHandler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestCORSHanyaOriginYangDiizinkan(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://evil.example")
	recorder := httptest.NewRecorder()
	testHandler().ServeHTTP(recorder, request)
	if value := recorder.Header().Get("Access-Control-Allow-Origin"); value != "" {
		t.Fatalf("origin tidak sah mendapat CORS: %s", value)
	}
}
