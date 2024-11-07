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
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/topfreegames/eventsgateway/v4/metrics"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc/keepalive"
	"strings"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/v4/logger"
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
	var err error

	err = c.initConfig(configPrefix)
	if err != nil {
		return nil, err
	}
	err = metrics.RegisterMetrics()
	if err != nil {
		return nil, err
	}

	clientKeepaliveTime := c.config.GetDuration(fmt.Sprintf("%sclient.keepalive.time", configPrefix))
	clientKeepaliveTimeout := c.config.GetDuration(fmt.Sprintf("%sclient.keepalive.timeout", configPrefix))
	clientKeepalivePermitWithoutStreams := c.config.GetBool(fmt.Sprintf("%sclient.keepalive.permitwithoutstreams", configPrefix))
	async := c.config.GetBool(fmt.Sprintf("%sclient.async", configPrefix))

	dialOpts := append(
		[]grpc.DialOption{
			grpc.WithChainUnaryInterceptor(
				otelgrpc.UnaryClientInterceptor(
					otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
					otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
				),
				otgrpc.OpenTracingClientInterceptor(opentracing.GlobalTracer()),
			),
			grpc.WithKeepaliveParams(
				keepalive.ClientParameters{
					Time:                clientKeepaliveTime,
					Timeout:             clientKeepaliveTimeout,
					PermitWithoutStream: clientKeepalivePermitWithoutStreams,
				}),
		},
		opts...,
	)

	c.logger = c.logger.WithFields(map[string]interface{}{
		"serverAddress": c.serverAddress,
		"async":         async,
		"source":        "eventsgateway/client",
		"topic":         c.topic,
	})

	if async {
		c.client, err = newGRPCClientAsync(configPrefix, c.config, c.logger, c.serverAddress, client, dialOpts...)
	} else {
		c.client, err = newGRPCClientSync(configPrefix, c.config, c.logger, c.serverAddress, client, dialOpts...)
	}

	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) initConfig(configPrefix string) error {

	keepaliveTime, _ := time.ParseDuration("60s")
	keepaliveTimeout, _ := time.ParseDuration("15s")
	lingerInterval, _ := time.ParseDuration("500ms")
	retryInterval, _ := time.ParseDuration("2s")

	topicConfigKey := fmt.Sprintf("%sclient.kafkatopic", configPrefix)
	c.topic = c.config.GetString(topicConfigKey)
	if c.topic == "" {
		return fmt.Errorf("no kafka topic informed at %s", topicConfigKey)
	}

	serverAddressKey := fmt.Sprintf("%sclient.grpc.serverAddress", configPrefix)
	c.serverAddress = c.config.GetString(serverAddressKey)
	if c.serverAddress == "" {
		return fmt.Errorf("no grpc server address informed at %s", serverAddressKey)
	}

	c.config.SetDefault(fmt.Sprintf("%sclient.keepalive.time", configPrefix), keepaliveTime)
	c.config.SetDefault(fmt.Sprintf("%sclient.keepalive.timeout", configPrefix), keepaliveTimeout)
	c.config.SetDefault(fmt.Sprintf("%sclient.keepalive.permitwithoutstreams", configPrefix), true)
	c.config.SetDefault(fmt.Sprintf("%sclient.channelBuffer", configPrefix), 500)
	c.config.SetDefault(fmt.Sprintf("%sclient.lingerInterval", configPrefix), lingerInterval)
	c.config.SetDefault(fmt.Sprintf("%sclient.batchSize", configPrefix), 50)
	c.config.SetDefault(fmt.Sprintf("%sclient.maxRetries", configPrefix), 3)
	c.config.SetDefault(fmt.Sprintf("%sclient.numRoutines", configPrefix), 2)
	c.config.SetDefault(fmt.Sprintf("%sclient.retryInterval", configPrefix), retryInterval)

	return nil
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
