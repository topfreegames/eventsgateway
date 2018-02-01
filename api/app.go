package api

import (
	"fmt"
	"net/http"

	"git.topfreegames.com/eventsgateway/forwarder"
	"git.topfreegames.com/eventsgateway/middleware"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// App is the api app structure
type App struct {
	host            string
	port            int
	log             logrus.FieldLogger
	Router          *mux.Router
	eventsForwarder forwarder.Forwarder
}

// NewApp creates a new App object
func NewApp(host string, port int, log logrus.FieldLogger) (*App, error) {
	a := &App{
		host: host,
		port: port,
		log:  log,
	}
	err := a.configure()
	return a, err
}

func (a *App) configure() error {
	err := a.configureEventsForwarder()
	if err != nil {
		return err
	}
	a.AddRoutes()
	return nil
}

func (a *App) configureEventsForwarder() error {
	return nil
}

// AddRoutes add routes to the router
func (a *App) AddRoutes() {
	a.Router = mux.NewRouter()

	a.Router.Handle("/healthcheck", middleware.Chain(
		NewHealthcheckHandler(),
		middleware.NewLoggingMiddleware(a.log),
	)).Name("healthcheck")

	a.Router.Handle("/api/v2/applications/{game}/uploads", middleware.Chain(
		NewEventsHandler(a.eventsForwarder),
		middleware.NewLoggingMiddleware(a.log),
	)).Methods("POST").Name("events")
}

// Run runs the app
func (a *App) Run() {
	log := a.log
	log.Infof("events gateway listening on %s:%d", a.host, a.port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", a.host, a.port), a.Router))
}
