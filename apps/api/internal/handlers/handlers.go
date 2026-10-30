package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/codesyariah/codenet-router/apps/api/internal/auth"
	"github.com/codesyariah/codenet-router/apps/api/internal/middleware"
	"github.com/codesyariah/codenet-router/apps/api/internal/providers"
	"github.com/codesyariah/codenet-router/apps/api/internal/services"
)

type API struct {
	service      *services.RouterService
	auth         *auth.Service
	logger       *slog.Logger
	cookieName   string
	cookieSecure bool
}
type response struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    any          `json:"data,omitempty"`
	Error   *errorDetail `json:"error,omitempty"`
}
type errorDetail struct {
	Code string `json:"code"`
}

type Config struct {
	AllowedOrigin string
	CookieName    string
	CookieSecure  bool
}

func New(service *services.RouterService, authService *auth.Service, logger *slog.Logger, config Config) http.Handler {
	if config.CookieName == "" {
		config.CookieName = "codenet_session"
	}
	api := &API{service: service, auth: authService, logger: logger, cookieName: config.CookieName, cookieSecure: config.CookieSecure}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("POST /api/v1/auth/login", api.login)
	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/v1/auth/session", api.session)
	protected.HandleFunc("POST /api/v1/auth/logout", api.logout)
	protected.HandleFunc("GET /api/v1/dashboard", api.dashboard)
	protected.HandleFunc("GET /api/v1/system/info", api.systemInfo)
	protected.HandleFunc("GET /api/v1/system/resources", api.resources)
	protected.HandleFunc("GET /api/v1/system/services", api.services)
	protected.HandleFunc("GET /api/v1/interfaces", api.interfaces)
	protected.HandleFunc("GET /api/v1/interfaces/{name}", api.interfaceByName)
	protected.HandleFunc("GET /api/v1/pppoe/status", api.pppoeStatus)
	protected.HandleFunc("GET /api/v1/pppoe/sessions", api.pppoeSessions)
	protected.HandleFunc("GET /api/v1/bandwidth/profiles", api.bandwidthProfiles)
	protected.HandleFunc("GET /api/v1/bandwidth/statistics", api.bandwidthStatistics)
	mux.Handle("/api/v1/", middleware.Authenticate(authService, config.CookieName, middleware.CSRFForUnsafeMethods(protected)))
	var handler http.Handler = mux
	handler = middleware.RequestTimeout(8*time.Second, handler)
	handler = middleware.ContentLength(handler)
	handler = middleware.CORS(config.AllowedOrigin, handler)
	handler = middleware.Security(handler)
	handler = middleware.Log(logger, handler)
	handler = middleware.Recover(logger, handler)
	return handler
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		a.writeError(w, http.StatusUnsupportedMediaType, "CONTENT_TYPE_REQUIRED", "Content-Type application/json diperlukan")
		return
	}
	var input loginRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, 16*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		a.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Format permintaan tidak valid")
		return
	}
	if auth.ValidateUsername(input.Username) != nil || input.Password == "" || len(input.Password) > 1024 {
		a.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Username atau password tidak valid")
		return
	}
	result, err := a.auth.Login(r.Context(), input.Username, input.Password, r.RemoteAddr)
	if errors.Is(err, auth.ErrRateLimited) {
		w.Header().Set("Retry-After", "900")
		a.writeError(w, http.StatusTooManyRequests, "LOGIN_RATE_LIMITED", "Terlalu banyak percobaan login; coba kembali nanti")
		return
	}
	if err != nil {
		a.writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Username atau password tidak valid")
		return
	}
	maxAge := int(time.Until(result.Session.ExpiresAt).Seconds())
	http.SetCookie(w, &http.Cookie{Name: a.cookieName, Value: result.RawToken, Path: "/", HttpOnly: true, Secure: a.cookieSecure, SameSite: http.SameSiteStrictMode, MaxAge: maxAge, Expires: result.Session.ExpiresAt})
	a.write(w, http.StatusOK, "Login berhasil", map[string]any{"admin": map[string]string{"username": result.Session.Admin.Username}, "csrf_token": result.Session.CSRFToken, "expires_at": result.Session.ExpiresAt})
}

func (a *API) session(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.SessionFromContext(r.Context())
	if !ok {
		a.writeError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Autentikasi diperlukan")
		return
	}
	a.write(w, http.StatusOK, "Session aktif", map[string]any{"admin": map[string]string{"username": session.Admin.Username}, "csrf_token": session.CSRFToken, "expires_at": session.ExpiresAt})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	session, ok := middleware.SessionFromContext(r.Context())
	if !ok {
		a.writeError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Autentikasi diperlukan")
		return
	}
	if err := a.auth.Logout(r.Context(), session.ID); err != nil {
		a.logger.Error("logout gagal", "error", err)
		a.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Logout gagal")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: a.cookieName, Value: "", Path: "/", HttpOnly: true, Secure: a.cookieSecure, SameSite: http.SameSiteStrictMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	a.write(w, http.StatusOK, "Logout berhasil", map[string]bool{"logged_out": true})
}

func (a *API) write(w http.ResponseWriter, status int, message string, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response{Success: true, Message: message, Data: data})
}
func (a *API) fail(w http.ResponseWriter, err error) {
	status, code, message := http.StatusInternalServerError, "INTERNAL_ERROR", "Data tidak dapat diambil"
	var notFound providers.NotFoundError
	if errors.As(err, &notFound) {
		status, code, message = http.StatusNotFound, "NOT_FOUND", err.Error()
	}
	a.logger.Error("provider gagal", "error", err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response{Success: false, Message: message, Error: &errorDetail{Code: code}})
}

func (a *API) writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response{Success: false, Message: message, Error: &errorDetail{Code: code}})
}
func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	a.write(w, http.StatusOK, "API siap", map[string]string{"status": "ok"})
}
func (a *API) dashboard(w http.ResponseWriter, r *http.Request) {
	data, err := a.service.Dashboard(r.Context())
	if err != nil {
		a.fail(w, err)
		return
	}
	a.write(w, http.StatusOK, "Dashboard berhasil diambil", data)
}
func (a *API) systemInfo(w http.ResponseWriter, r *http.Request) {
	data, err := a.service.Provider().SystemInfo(r.Context())
	if err != nil {
		a.fail(w, err)
		return
	}
	a.write(w, 200, "Informasi sistem berhasil diambil", data)
}
func (a *API) resources(w http.ResponseWriter, r *http.Request) {
	data, err := a.service.Provider().ResourceUsage(r.Context())
	if err != nil {
		a.fail(w, err)
		return
	}
	a.write(w, 200, "Penggunaan resource berhasil diambil", data)
}
func (a *API) services(w http.ResponseWriter, r *http.Request) {
	data, err := a.service.Provider().Services(r.Context())
	if err != nil {
		a.fail(w, err)
		return
	}
	a.write(w, 200, "Status layanan berhasil diambil", data)
}
func (a *API) interfaces(w http.ResponseWriter, r *http.Request) {
	data, err := a.service.Provider().Interfaces(r.Context())
	if err != nil {
		a.fail(w, err)
		return
	}
	a.write(w, 200, "Daftar interface berhasil diambil", data)
}
func (a *API) interfaceByName(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !validInterfaceName(name) {
		a.writeError(w, http.StatusBadRequest, "INVALID_INTERFACE", "Nama interface tidak valid")
		return
	}
	data, err := a.service.Provider().Interface(r.Context(), name)
	if err != nil {
		a.fail(w, err)
		return
	}
	a.write(w, 200, "Interface berhasil diambil", data)
}

func validInterfaceName(name string) bool {
	if name == "" || len(name) > 15 || strings.ContainsAny(name, "/\\\x00") {
		return false
	}
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || strings.ContainsRune("_.:-", char)) {
			return false
		}
	}
	return true
}
func (a *API) pppoeStatus(w http.ResponseWriter, r *http.Request) {
	data, err := a.service.Provider().PPPoEStatus(r.Context())
	if err != nil {
		a.fail(w, err)
		return
	}
	a.write(w, 200, "Status PPPoE berhasil diambil", data)
}
func (a *API) pppoeSessions(w http.ResponseWriter, r *http.Request) {
	data, err := a.service.Provider().PPPoESessions(r.Context())
	if err != nil {
		a.fail(w, err)
		return
	}
	a.write(w, 200, "Sesi PPPoE berhasil diambil", data)
}
func (a *API) bandwidthProfiles(w http.ResponseWriter, r *http.Request) {
	data, err := a.service.Provider().BandwidthProfiles(r.Context())
	if err != nil {
		a.fail(w, err)
		return
	}
	a.write(w, 200, "Profil bandwidth berhasil diambil", data)
}
func (a *API) bandwidthStatistics(w http.ResponseWriter, r *http.Request) {
	data, err := a.service.Provider().BandwidthStatistics(r.Context())
	if err != nil {
		a.fail(w, err)
		return
	}
	a.write(w, 200, "Statistik bandwidth berhasil diambil", data)
}
