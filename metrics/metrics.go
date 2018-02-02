// MIT License
//
// Copyright (c) 2018 Top Free Games
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package metrics

import (
	"fmt"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// APIResponseTime summary
	APIResponseTime = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "eventsgateway",
			Subsystem:  "api",
			Name:       "response_time_ms",
			Help:       "the response time in ms of api routes",
			Objectives: map[float64]float64{0.7: 0.02, 0.95: 0.005, 0.99: 0.001},
		},
		[]string{"hostname", "route", "topic"},
	)

	// APIRequestsSuccessCounter counter
	APIRequestsSuccessCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "eventsgateway",
			Subsystem: "api",
			Name:      "requests_success_counter",
			Help:      "A counter of succeeded api requests",
		},
		[]string{"hostname", "route", "topic"},
	)

	// APIRequestsFailureCounter counter
	APIRequestsFailureCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "eventsgateway",
			Subsystem: "api",
			Name:      "requests_failure_counter",
			Help:      "A counter of failed api requests",
		},
		[]string{"hostname", "route", "topic", "reason"},
	)

	// ClientRequestsResponseTime is the time the client take to talk to the server
	ClientRequestsResponseTime = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "eventsgateway",
		Subsystem:  "client",
		Name:       "response_time_ms",
		Help:       "the response time in ms of calls to server",
		Objectives: map[float64]float64{0.7: 0.02, 0.95: 0.005, 0.99: 0.001},
	},
		[]string{"clientHost", "route", "topic"},
	)

	// ClientRequestsSuccessCounter is the count of successfull calls to the server
	ClientRequestsSuccessCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "eventsgateway",
		Subsystem: "client",
		Name:      "requests_success_counter",
		Help:      "the count of successfull client requests to the server",
	},
		[]string{"clientHost", "route", "topic"},
	)

	// ClientRequestsFailureCounter is the count of failed calls to the server
	ClientRequestsFailureCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "eventsgateway",
		Subsystem: "client",
		Name:      "requests_failure_counter",
		Help:      "the count of failed client requests to the server",
	},
		[]string{"clientHost", "route", "topic", "reason"},
	)
)

func init() {
	prometheus.MustRegister(
		APIResponseTime,
		APIRequestsFailureCounter,
		APIRequestsSuccessCounter,
		ClientRequestsResponseTime,
		ClientRequestsSuccessCounter,
		ClientRequestsFailureCounter,
	)
	port := ":9090"
	if envPort, ok := os.LookupEnv("EVENTSGATEWAY_PROMETHEUS_PORT"); ok {
		port = fmt.Sprintf(":%s", envPort)
	}
	go func() {
		r := mux.NewRouter()
		r.Handle("/metrics", promhttp.Handler())

		s := &http.Server{
			Addr:           port,
			ReadTimeout:    8 * time.Second,
			WriteTimeout:   8 * time.Second,
			MaxHeaderBytes: 1 << 20,
			Handler:        r,
		}
		log.Fatal(s.ListenAndServe())
	}()
}
