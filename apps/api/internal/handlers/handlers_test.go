package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codesyariah/codenet-router/apps/api/internal/auth"
	"github.com/codesyariah/codenet-router/apps/api/internal/models"
	"github.com/codesyariah/codenet-router/apps/api/internal/providers"
	"github.com/codesyariah/codenet-router/apps/api/internal/services"
)

const testPassword = "Correct-Horse-Battery-99!"

func testHandler(t *testing.T, authConfig auth.Config, secureCookie bool) http.Handler {
	t.Helper()
	return testHandlerWithProvider(t, authConfig, secureCookie, providers.NewMock())
}

func testHandlerWithProvider(t *testing.T, authConfig auth.Config, secureCookie bool, provider providers.RouterProvider) http.Handler {
	t.Helper()
	store, err := auth.OpenStore(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.CreateAdmin(context.Background(), "admin", testPassword); err != nil {
		t.Fatal(err)
	}
	authService := auth.NewService(store, authConfig)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(services.NewRouter(provider), authService, logger, Config{AllowedOrigin: "http://127.0.0.1:5173", CookieName: "codenet_session", CookieSecure: secureCookie})
}

func perform(handler http.Handler, method, path string, body []byte, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.RemoteAddr = "192.0.2.10:12345"
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	if csrf != "" {
		request.Header.Set("X-CSRF-Token", csrf)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func login(t *testing.T, handler http.Handler) (*http.Cookie, string) {
	t.Helper()
	payload, _ := json.Marshal(map[string]string{"username": "admin", "password": testPassword})
	recorder := perform(handler, http.MethodPost, "/api/v1/auth/login", payload, nil, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("login gagal: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	result := struct {
		Data struct {
			CSRFToken string `json:"csrf_token"`
		} `json:"data"`
	}{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("mengharapkan satu cookie, mendapat %d", len(cookies))
	}
	return cookies[0], result.Data.CSRFToken
}

func TestHealthTetapPublik(t *testing.T) {
	recorder := perform(testHandler(t, auth.Config{}, false), http.MethodGet, "/health", nil, nil, "")
	if recorder.Code != http.StatusOK || strings.Contains(recorder.Body.String(), "hostname") {
		t.Fatalf("health tidak minimal: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestEndpointMonitoringMemerlukanAutentikasi(t *testing.T) {
	recorder := perform(testHandler(t, auth.Config{}, false), http.MethodGet, "/api/v1/dashboard", nil, nil, "")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestLoginBerhasilDanCookieAman(t *testing.T) {
	handler := testHandler(t, auth.Config{}, true)
	cookie, csrf := login(t, handler)
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.Value == "" {
		t.Fatalf("atribut cookie tidak aman: %#v", cookie)
	}
	if csrf == "" {
		t.Fatal("token CSRF tidak tersedia")
	}
	recorder := perform(handler, http.MethodGet, "/api/v1/dashboard", nil, cookie, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d", recorder.Code)
	}
}

func TestLoginGagal(t *testing.T) {
	handler := testHandler(t, auth.Config{}, false)
	recorder := perform(handler, http.MethodPost, "/api/v1/auth/login", []byte(`{"username":"admin","password":"salah"}`), nil, "")
	if recorder.Code != http.StatusUnauthorized || !strings.Contains(recorder.Body.String(), "INVALID_CREDENTIALS") {
		t.Fatalf("respons login gagal salah: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestRateLimitLogin(t *testing.T) {
	handler := testHandler(t, auth.Config{LoginMaxAttempts: 2, LoginWindow: time.Hour}, false)
	payload := []byte(`{"username":"admin","password":"salah"}`)
	for i := 0; i < 2; i++ {
		if code := perform(handler, http.MethodPost, "/api/v1/auth/login", payload, nil, "").Code; code != http.StatusUnauthorized {
			t.Fatalf("percobaan %d status=%d", i+1, code)
		}
	}
	recorder := perform(handler, http.MethodPost, "/api/v1/auth/login", payload, nil, "")
	if recorder.Code != http.StatusTooManyRequests || recorder.Header().Get("Retry-After") == "" {
		t.Fatalf("rate limit tidak aktif: %d", recorder.Code)
	}
}

func TestSessionExpiration(t *testing.T) {
	handler := testHandler(t, auth.Config{SessionTTL: time.Millisecond}, false)
	cookie, _ := login(t, handler)
	time.Sleep(5 * time.Millisecond)
	recorder := perform(handler, http.MethodGet, "/api/v1/auth/session", nil, cookie, "")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("session seharusnya expired: %d", recorder.Code)
	}
}

func TestLogoutDanCSRF(t *testing.T) {
	handler := testHandler(t, auth.Config{}, false)
	cookie, csrf := login(t, handler)
	if code := perform(handler, http.MethodPost, "/api/v1/auth/logout", nil, cookie, "token-salah").Code; code != http.StatusForbidden {
		t.Fatalf("CSRF invalid status=%d", code)
	}
	if code := perform(handler, http.MethodPost, "/api/v1/auth/logout", nil, cookie, csrf).Code; code != http.StatusOK {
		t.Fatalf("logout status=%d", code)
	}
	if code := perform(handler, http.MethodGet, "/api/v1/auth/session", nil, cookie, "").Code; code != http.StatusUnauthorized {
		t.Fatalf("session masih aktif setelah logout: %d", code)
	}
}

func TestValidasiInputLoginDanInterface(t *testing.T) {
	handler := testHandler(t, auth.Config{}, false)
	invalidLogin := perform(handler, http.MethodPost, "/api/v1/auth/login", []byte(`{"username":"a","password":"x","extra":true}`), nil, "")
	if invalidLogin.Code != http.StatusBadRequest {
		t.Fatalf("login invalid status=%d", invalidLogin.Code)
	}
	cookie, _ := login(t, handler)
	invalidInterface := perform(handler, http.MethodGet, "/api/v1/interfaces/bad$name", nil, cookie, "")
	if invalidInterface.Code != http.StatusBadRequest {
		t.Fatalf("interface invalid status=%d", invalidInterface.Code)
	}
}

func TestErrorProviderTidakMembocorkanDetail(t *testing.T) {
	provider := &failingProvider{RouterProvider: providers.NewMock()}
	handler := testHandlerWithProvider(t, auth.Config{}, false, provider)
	cookie, _ := login(t, handler)
	recorder := perform(handler, http.MethodGet, "/api/v1/interfaces", nil, cookie, "")
	if recorder.Code != http.StatusInternalServerError || strings.Contains(recorder.Body.String(), "detail internal") {
		t.Fatalf("error handling tidak aman: %d %s", recorder.Code, recorder.Body.String())
	}
}

type failingProvider struct{ providers.RouterProvider }

func (f *failingProvider) Interfaces(context.Context) ([]models.Interface, error) {
	return nil, errors.New("detail internal provider")
}
