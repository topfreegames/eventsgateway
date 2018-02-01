package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

//LoggingMiddleware handles logging
type LoggingMiddleware struct {
	log  logrus.FieldLogger
	next http.Handler
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(log logrus.FieldLogger) *LoggingMiddleware {
	m := &LoggingMiddleware{log: log}
	return m
}

const requestIDKey = contextKey("requestID")
const loggerKey = contextKey("logger")

func newContextWithRequestIDAndLogger(ctx context.Context, logger logrus.FieldLogger) context.Context {
	reqID := uuid.NewV4().String()
	l := logger.WithField("requestID", reqID)

	c := context.WithValue(ctx, requestIDKey, reqID)
	c = context.WithValue(c, loggerKey, l)
	return c
}

// LoggerFromContext grabs the current logger
func LoggerFromContext(ctx context.Context) logrus.FieldLogger {
	return ctx.Value(loggerKey).(logrus.FieldLogger)
}

// ServeHTTP method
func (m *LoggingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := newContextWithRequestIDAndLogger(r.Context(), m.log)

	start := time.Now()
	defer func() {
		l := LoggerFromContext(ctx)
		status := getStatusFromResponseWriter(w)
		route, _ := mux.CurrentRoute(r).GetPathTemplate()
		// request failed
		if status > 399 && status < 500 {
			l.WithFields(logrus.Fields{
				"path":            r.URL.Path,
				"route":           route,
				"requestDuration": time.Since(start).Nanoseconds(),
				"status":          status,
			}).Warn("Request failed.")
		} else if status > 499 { // request is ok, but server failed
			l.WithFields(logrus.Fields{
				"path":            r.URL.Path,
				"route":           route,
				"requestDuration": time.Since(start).Nanoseconds(),
				"status":          status,
			}).Error("Response failed.")
		} else { // Everything went ok
			l.WithFields(logrus.Fields{
				"path":            r.URL.Path,
				"route":           route,
				"requestDuration": time.Since(start).Nanoseconds(),
				"status":          status,
			}).Debug("Request successful.")
		}
	}()

	m.next.ServeHTTP(w, r.WithContext(ctx))
}

func getStatusFromResponseWriter(w http.ResponseWriter) int {
	rw, ok := w.(*responseWriter)
	if ok {
		return rw.statusCode
	}
	return -1
}

//SetNext middleware
func (m *LoggingMiddleware) SetNext(next http.Handler) {
	m.next = next
}
