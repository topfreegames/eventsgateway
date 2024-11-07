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
	"github.com/sirupsen/logrus"
)

const metricsNamespace = "eventsgateway"
const metricsSubsystem = "client_v4"

const (
	// LabelRoute is the GRPC route the request is reaching
	LabelRoute = "route"
	// LabelTopic is the Kafka topic the event refers to
	LabelTopic = "topic"
	// LabelStatus is the status of the request. OK if success or ERROR if fail
	LabelStatus = "status"
	// LabelRetry is the counter of the requests retries to EG server
	LabelRetry = "retry"
)

var (

	// ClientRequestsResponseTime summary, observes the API response time as perceived by the client
	ClientRequestsResponseTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "response_time_ms",
			Help:      "the response time in ms of calls to server",
			Buckets:   []float64{10, 30, 50, 100, 500},
		},
		[]string{LabelRoute, LabelTopic, LabelRetry, LabelStatus},
	)

	// ClientEventsCounter is the count of events broken by topic and status
	ClientEventsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Subsystem: metricsSubsystem,
		Name:      "events_counter",
		Help:      "the count of successfull client requests to the server",
	},
		[]string{LabelRoute, LabelTopic, LabelStatus},
	)

	// AsyncClientEventsChannelLength is the number of current events in the eventsChannel buffer
	AsyncClientEventsChannelLength = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Subsystem: metricsSubsystem,
		Name:      "async_events_channel_length",
		Help:      "the number of current events in the eventsChannel buffer in the async client",
	},
		[]string{LabelTopic},
	)

	// AsyncClientEventsDroppedCounter is the count of requests that were dropped due
	// to req.Retry > maxRetries. Only available for Async mode
	AsyncClientEventsDroppedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Subsystem: metricsSubsystem,
		Name:      "async_events_dropped_counter",
		Help:      "the count of dropped client async requests to the server",
	},
		[]string{LabelTopic},
	)
)

// RegisterMetrics is a wrapper to handle prometheus.AlreadyRegisteredError;
// it only returns an error if the metric wasn't already registered and there was an
// actual error registering it.
func RegisterMetrics() error {

	collectors := []prometheus.Collector{
		ClientRequestsResponseTime,
		ClientEventsCounter,
		AsyncClientEventsDroppedCounter,
		AsyncClientEventsChannelLength,
	}

	for _, collector := range collectors {
		err := prometheus.Register(collector)
		if err != nil {
			logrus.New().Warnf("Error while registering EG metrics: %s", err)
			var alreadyRegisteredError prometheus.AlreadyRegisteredError
			if !errors.As(err, &alreadyRegisteredError) {
				return err
			}
		}
	}

	return nil
}
