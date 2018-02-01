package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"git.topfreegames.com/eventsgateway/forwarder"
	"git.topfreegames.com/eventsgateway/middleware"
)

//EventsHandler handler
type EventsHandler struct {
	eventsForwarder forwarder.Forwarder
}

// NewEventsHandler creates a new events handler
func NewEventsHandler(eventsForwarder forwarder.Forwarder) *EventsHandler {
	h := &EventsHandler{
		eventsForwarder: eventsForwarder,
	}
	return h
}

//ServeHTTP method
func (h *EventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.LoggerFromContext(r.Context())
	partition := mux.Vars(r)["game"]
	logger.Debugf("events called with partition: %s", partition)

	Write(w, http.StatusOK, fmt.Sprintf(`{"status": "ok"}`))
}
