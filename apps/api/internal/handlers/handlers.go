package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/codesyariah/codenet-router/apps/api/internal/middleware"
	"github.com/codesyariah/codenet-router/apps/api/internal/providers"
	"github.com/codesyariah/codenet-router/apps/api/internal/services"
)

type API struct {
	service *services.RouterService
	logger  *slog.Logger
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

func New(service *services.RouterService, logger *slog.Logger, allowedOrigin string) http.Handler {
	api := &API{service: service, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("GET /api/v1/dashboard", api.dashboard)
	mux.HandleFunc("GET /api/v1/system/info", api.systemInfo)
	mux.HandleFunc("GET /api/v1/system/resources", api.resources)
	mux.HandleFunc("GET /api/v1/system/services", api.services)
	mux.HandleFunc("GET /api/v1/interfaces", api.interfaces)
	mux.HandleFunc("GET /api/v1/interfaces/{name}", api.interfaceByName)
	mux.HandleFunc("GET /api/v1/pppoe/status", api.pppoeStatus)
	mux.HandleFunc("GET /api/v1/pppoe/sessions", api.pppoeSessions)
	mux.HandleFunc("GET /api/v1/bandwidth/profiles", api.bandwidthProfiles)
	mux.HandleFunc("GET /api/v1/bandwidth/statistics", api.bandwidthStatistics)
	var handler http.Handler = mux
	handler = middleware.ContentLength(handler)
	handler = middleware.CORS(allowedOrigin, handler)
	handler = middleware.Security(handler)
	handler = middleware.Log(logger, handler)
	handler = middleware.Recover(logger, handler)
	return handler
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
	if name == "" || strings.ContainsAny(name, "/\\\x00") {
		a.fail(w, providers.NotFoundError{Resource: "interface"})
		return
	}
	data, err := a.service.Provider().Interface(r.Context(), name)
	if err != nil {
		a.fail(w, err)
		return
	}
	a.write(w, 200, "Interface berhasil diambil", data)
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
