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
	"errors"
	"net/http"
	"time"

	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// LabelRoute is the GRPC route the request is reaching
	LabelRoute = "route"
	// LabelTopic is the Kafka topic the event refers to
	LabelTopic = "topic"
	// LabelStatus is the status of the request. OK if success or ERROR if fail
	LabelStatus = "status"
)

var (
	// APIResponseTime summary, observes the API response time as perceived by the server
	APIResponseTime *prometheus.HistogramVec

	// APIPayloadSize summary, observes the payload size of requests arriving at the server
	APIPayloadSize *prometheus.HistogramVec

	// KafkaRequestLatency summary, observes that kafka request latency per topic and status
	KafkaRequestLatency *prometheus.HistogramVec
)

func defaultLatencyBuckets(config *viper.Viper) []float64 {
	// in milliseconds
	const configKey = "prometheus.buckets.latency"
	config.SetDefault(configKey, []float64{10, 30, 50, 100, 500})
	return config.Get(configKey).([]float64)
}

func defaultPayloadSizeBuckets(config *viper.Viper) []float64 {
	// in bytes
	configKey := "prometheus.buckets.payloadSize"
	config.SetDefault(configKey, []float64{10000, 50000, 100000, 500000, 1000000, 5000000})
	return config.Get(configKey).([]float64)
}

// RegisterMetrics is a wrapper to handle prometheus.AlreadyRegisteredError;
// it only returns an error if the metric wasn't already registered and there was an
// actual error registering it.
func RegisterMetrics(collectors []prometheus.Collector) error {
	for _, collector := range collectors {
		err := prometheus.Register(collector)
		if err != nil {
			var alreadyRegisteredError prometheus.AlreadyRegisteredError
			if !errors.As(err, &alreadyRegisteredError) {
				return err
			}
		}
	}
	return nil
}

// StartServer runs a metrics server inside a goroutine
// that reports default application metrics in prometheus format.
// Any errors that may occur will stop the server and log.Fatal the error.
func StartServer(config *viper.Viper) {
	APIPayloadSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "eventsgateway",
			Subsystem: "api",
			Name:      "payload_size",
			Help:      "payload size of API routes, in bytes",
			Buckets:   defaultPayloadSizeBuckets(config),
		},
		[]string{LabelTopic},
	)

	APIResponseTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "eventsgateway",
			Subsystem: "api",
			Name:      "response_time_ms",
			Help:      "the response time in ms of api routes",
			Buckets:   defaultLatencyBuckets(config),
		},
		[]string{LabelRoute, LabelStatus, LabelTopic},
	)

	KafkaRequestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "eventsgateway",
			Subsystem: "kafka",
			Name:      "response_time_ms",
			Help:      "the response time in ms of Kafka",
			Buckets:   defaultLatencyBuckets(config),
		},
		[]string{LabelStatus, LabelTopic},
	)

	collectors := []prometheus.Collector{
		APIResponseTime,
		APIPayloadSize,
		KafkaRequestLatency,
	}

	err := RegisterMetrics(collectors)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		envEnabled := config.GetString("prometheus.enabled")
		if envEnabled != "true" {
			log.Warn("Prometheus web server disabled")
			return
		}

		r := mux.NewRouter()
		r.Handle("/metrics", promhttp.Handler())

		s := &http.Server{
			Addr:           config.GetString("prometheus.port"),
			ReadTimeout:    8 * time.Second,
			WriteTimeout:   8 * time.Second,
			MaxHeaderBytes: 1 << 20,
			Handler:        r,
		}
		log.Fatal(s.ListenAndServe())
	}()
}
