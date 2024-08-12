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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
)

const metricsNamespace = "eventsgateway"
const metricsSubsystem = "client"

var (

	// ClientRequestsResponseTime summary, observes the API response time as perceived by the client
	ClientRequestsResponseTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "response_time_ms",
			Help:      "the response time in ms of calls to server",
			Buckets:   []float64{3, 5, 10, 50, 100, 300, 500, 1000, 5000},
		},
		[]string{"route", "topic", "retry"},
	)

	// ClientRequestsSuccessCounter is the count of successfull calls to the server
	ClientRequestsSuccessCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Subsystem: metricsSubsystem,
		Name:      "requests_success_counter",
		Help:      "the count of successfull client requests to the server",
	},
		[]string{"route", "topic", "retry"},
	)

	// ClientRequestsFailureCounter is the count of failed calls to the server
	ClientRequestsFailureCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Subsystem: metricsSubsystem,
		Name:      "requests_failure_counter",
		Help:      "the count of failed client requests to the server",
	},
		[]string{"route", "topic", "retry", "reason"},
	)

	// AsyncClientRequestsDroppedCounter is the count of requests that were dropped due
	// to req.Retry > maxRetries. Only available for Async mode
	AsyncClientRequestsDroppedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Subsystem: metricsSubsystem,
		Name:      "async_requests_dropped_counter",
		Help:      "the count of dropped client async requests to the server",
	},
		[]string{"topic"},
	)
)

// RegisterMetrics is a wrapper to handle prometheus.AlreadyRegisteredError;
// it only returns an error if the metric wasn't already registered and there was an
// actual error registering it.
func RegisterMetrics(configPrefix string, config *viper.Viper) error {
	collectors := []prometheus.Collector{
		ClientRequestsResponseTime,
		ClientRequestsSuccessCounter,
		ClientRequestsFailureCounter,
		AsyncClientRequestsDroppedCounter,
	}

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
