// eventsgateway
// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2018 Top Free Games <backend@tfgco.com>

package client

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/topfreegames/eventsgateway/v4/metrics"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"strings"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/v4/logger"
	logruswrapper "github.com/topfreegames/eventsgateway/v4/logger/logrus"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
	"google.golang.org/grpc"
)

// Client struct
type Client struct {
	client        GRPCClient
	config        *viper.Viper
	logger        logger.Logger
	topic         string
	wg            sync.WaitGroup
	serverAddress string
}

// NewClient ctor (DEPRECATED, use New() instead)
// configPrefix is whatever comes before `client` subpart of config
func NewClient(
	configPrefix string,
	config *viper.Viper,
	logger logrus.FieldLogger,
	client pb.GRPCForwarderClient,
	opts ...grpc.DialOption,
) (*Client, error) {
	return New(configPrefix, config, logruswrapper.NewWithLogger(logger), client, opts...)
}

// New ctor
// configPrefix is whatever comes before `client` subpart of config
func New(
	configPrefix string,
	config *viper.Viper,
	logger logger.Logger,
	client pb.GRPCForwarderClient,
	opts ...grpc.DialOption,
) (*Client, error) {
	if configPrefix != "" && !strings.HasSuffix(configPrefix, ".") {
		configPrefix = strings.Join([]string{configPrefix, "."}, "")
	}
	c := &Client{
		config: config,
		logger: logger,
	}
	topicConf := fmt.Sprintf("%sclient.kafkatopic", configPrefix)
	c.topic = c.config.GetString(topicConf)
	if c.topic == "" {
		return nil, fmt.Errorf("no kafka topic informed at %s", topicConf)
	}
	c.logger = c.logger.WithFields(map[string]interface{}{
		"source": "eventsgateway/client",
		"topic":  c.topic,
	})
	var err error

	dialOpts := append(
		[]grpc.DialOption{
			grpc.WithChainUnaryInterceptor(
				otelgrpc.UnaryClientInterceptor(
					otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
					otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
				),
			),
		},
		opts...,
	)

	if c.client, err = c.newGRPCClient(configPrefix, client, dialOpts...); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) newGRPCClient(
	configPrefix string,
	client pb.GRPCForwarderClient,
	opts ...grpc.DialOption,
) (GRPCClient, error) {
	serverConf := fmt.Sprintf("%sclient.grpc.serverAddress", configPrefix)
	c.serverAddress = c.config.GetString(serverConf)
	if c.serverAddress == "" {
		return nil, fmt.Errorf("no grpc server address informed at %s", serverConf)
	}
	asyncConf := fmt.Sprintf("%sclient.async", configPrefix)
	c.config.SetDefault(asyncConf, false)
	async := c.config.GetBool(asyncConf)
	c.logger = c.logger.WithFields(map[string]interface{}{
		"serverAddress": c.serverAddress,
		"async":         async,
	})

	c.registerMetrics(configPrefix)
	if async {
		return newGRPCClientAsync(configPrefix, c.config, c.logger, c.serverAddress, client, opts...)
	}
	return newGRPCClientSync(configPrefix, c.config, c.logger, c.serverAddress, client, opts...)
}

func (c *Client) registerMetrics(configPrefix string) {
	latencyBucketsConf := fmt.Sprintf("%sclient.prometheus.buckets.latency", configPrefix)
	c.config.SetDefault(latencyBucketsConf, []float64{3, 5, 10, 50, 100, 300, 500, 1000, 5000})
	latencyBuckets := c.config.Get(latencyBucketsConf).([]float64)

	metrics.ClientRequestsResponseTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "eventsgateway",
			Subsystem: "client",
			Name:      "response_time_ms",
			Help:      "the response time in ms of calls to server",
			Buckets:   latencyBuckets,
		},
		[]string{"route", "topic", "retry"},
	)

	collectors := []prometheus.Collector{
		metrics.ClientRequestsResponseTime,
		metrics.ClientRequestsSuccessCounter,
		metrics.ClientRequestsFailureCounter,
		metrics.ClientRequestsDroppedCounter,
	}
	err := metrics.RegisterMetrics(collectors)
	if err != nil {
		c.logger.WithError(err).Error("failed to register metric")
	}
}

// Send sends an event to another server via grpc using the client's configured topic
func (c *Client) Send(
	ctx context.Context,
	name string,
	props map[string]string,
) error {
	l := c.logger.WithFields(map[string]interface{}{
		"operation": "send",
		"event":     name,
	})
	l.Debug("sending event")
	if err := c.client.send(ctx, buildEvent(name, props, c.topic, time.Now())); err != nil {
		l.WithError(err).Error("send event failed")
		return err
	}
	return nil
}

// SendToTopic sends an event to another server via grpc using an explicit topic
func (c *Client) SendToTopic(
	ctx context.Context,
	name string,
	props map[string]string,
	topic string,
) error {
	l := c.logger.WithFields(map[string]interface{}{
		"operation": "sendToTopic",
		"event":     name,
		"topic":     topic,
	})
	l.Debug("sending event")
	if err := c.client.send(ctx, buildEvent(name, props, topic, time.Now())); err != nil {
		l.WithError(err).Error("send event failed")
		return err
	}
	return nil
}

// SendAtTime sends an event to another server via grpc with a specific timestamp
func (c *Client) SendAtTime(
	ctx context.Context,
	name string,
	props map[string]string,
	time time.Time,
) error {
	l := c.logger.WithFields(logrus.Fields{
		"operation": "sendAtTime",
		"event":     name,
		"time":      time,
	})
	l.Debug("sending event")
	if err := c.client.send(ctx, buildEvent(name, props, c.topic, time)); err != nil {
		l.WithError(err).Error("send event failed")
		return err
	}
	return nil
}

func (c *Client) GetGRPCClient() GRPCClient {
	return c.client
}

// GracefulStop waits pending async send of events and closes client connection
func (c *Client) GracefulStop() error {
	return c.client.GracefulStop()
}

func buildEvent(name string, props map[string]string, topic string, time time.Time) *pb.Event {
	return &pb.Event{
		Id:        uuid.NewV4().String(),
		Name:      name,
		Topic:     topic,
		Props:     props,
		Timestamp: time.UnixNano() / 1000000,
	}
}
