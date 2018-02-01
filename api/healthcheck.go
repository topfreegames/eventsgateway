package api

import (
	"net/http"

	"git.topfreegames.com/eventsgateway/middleware"
)

//HealthcheckHandler handler
type HealthcheckHandler struct {
}

// NewHealthcheckHandler creates a new healthcheck handler
func NewHealthcheckHandler() *HealthcheckHandler {
	m := &HealthcheckHandler{}
	return m
}

//ServeHTTP method
func (h *HealthcheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.LoggerFromContext(r.Context())

	logger.Debug("healthcheck called")

	Write(w, http.StatusOK, `{"healthy": true}`)
}
