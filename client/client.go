// eventsgateway
// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2018 Top Free Games <backend@tfgco.com>

package client

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/metrics"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	hostname, _ = os.Hostname()
)

// GRPCClient struct
type GRPCClient struct {
	async                 bool
	asyncRetryInterval    time.Duration
	asyncRetries          int
	client                pb.GRPCForwarderClient
	config                *viper.Viper
	numSendEventsRoutines int
	eventsChannel         chan *pb.Event
	flushInterval         time.Duration
	flushSize             int
	logger                log.FieldLogger
	serverAddress         string
	timeout               time.Duration
	topic                 string
	wg                    sync.WaitGroup
}

// NewClient returns a new GRPCClient
func NewClient(
	configPrefix string,
	config *viper.Viper,
	logger log.FieldLogger,
	client pb.GRPCForwarderClient,
) (*GRPCClient, error) {
	g := &GRPCClient{
		config: config,
		logger: logger,
	}
	if configPrefix != "" && !strings.HasSuffix(configPrefix, ".") {
		configPrefix = strings.Join([]string{configPrefix, "."}, "")
	}
	err := g.configure(configPrefix, client)
	if err != nil {
		g.logger.WithError(err).Error("failed to configure client")
		return nil, err
	}
	if g.async {
		for i := 0; i < g.numSendEventsRoutines; i++ {
			go g.sendEventsRoutine()
		}
	}
	return g, nil
}

func (g *GRPCClient) configure(configPrefix string, client pb.GRPCForwarderClient) error {
	if err := g.configureSend(configPrefix); err != nil {
		return err
	}
	if err := g.configureGRPC(configPrefix, client); err != nil {
		return err
	}
	return nil
}

func (g *GRPCClient) configureSend(configPrefix string) error {
	asyncConf := fmt.Sprintf("%sclient.send.async", configPrefix)
	g.config.SetDefault(asyncConf, false)
	g.async = g.config.GetBool(asyncConf)

	asyncRetriesConf := fmt.Sprintf("%sclient.send.asyncRetries", configPrefix)
	g.config.SetDefault(asyncRetriesConf, false)
	g.asyncRetries = g.config.GetInt(asyncRetriesConf)

	asyncRetryIntervalConf := fmt.Sprintf("%sclient.send.asyncRetryInterval", configPrefix)
	g.config.SetDefault(asyncRetryIntervalConf, false)
	g.asyncRetryInterval = g.config.GetDuration(asyncRetryIntervalConf)

	flushIntervalConf := fmt.Sprintf("%sclient.send.flushInterval", configPrefix)
	g.config.SetDefault(flushIntervalConf, 500*time.Millisecond)
	g.flushInterval = g.config.GetDuration(flushIntervalConf)

	flushSizeConf := fmt.Sprintf("%sclient.send.flushSize", configPrefix)
	g.config.SetDefault(flushSizeConf, 50)
	g.flushSize = g.config.GetInt(flushSizeConf)

	numRoutinesConf := fmt.Sprintf("%sclient.send.numRoutines", configPrefix)
	g.config.SetDefault(numRoutinesConf, 5)
	g.numSendEventsRoutines = g.config.GetInt(numRoutinesConf)

	channelBufferConf := fmt.Sprintf("%sclient.send.channelBuffer", configPrefix)
	g.config.SetDefault(channelBufferConf, 200)
	channelBuffer := g.config.GetInt(channelBufferConf)
	g.eventsChannel = make(chan *pb.Event, channelBuffer)

	g.logger = g.logger.WithFields(log.Fields{
		"async":         g.async,
		"flushInterval": g.flushInterval,
		"flushSize":     g.flushSize,
	})

	return nil
}

func (g *GRPCClient) configureGRPC(configPrefix string, client pb.GRPCForwarderClient) error {
	topicConf := fmt.Sprintf("%sclient.kafkatopic", configPrefix)
	g.topic = g.config.GetString(topicConf)
	if g.topic == "" {
		return fmt.Errorf("no kafka topic informed at %s", topicConf)
	}

	serverConf := fmt.Sprintf("%sclient.grpc.serverAddress", configPrefix)
	g.serverAddress = g.config.GetString(serverConf)
	if g.serverAddress == "" {
		return fmt.Errorf("no grpc server address informed at %s", serverConf)
	}

	timeoutConf := fmt.Sprintf("%sclient.grpc.timeout", configPrefix)
	g.config.SetDefault(timeoutConf, 500*time.Millisecond)
	g.timeout = g.config.GetDuration(timeoutConf)

	g.logger = g.logger.WithFields(log.Fields{
		"source":     "eventsgateway/client",
		"topic":      g.topic,
		"serverAddr": g.serverAddress,
		"timeout":    g.timeout,
	})

	if client != nil {
		g.client = client
		return nil
	}

	g.logger.WithFields(log.Fields{
		"operation":    "configure",
		"serverAdress": g.serverAddress,
	}).Info("connecting to grpc server")
	conn, err := grpc.Dial(
		g.serverAddress,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(g.metricsReporterInterceptor),
	)
	if err != nil {
		return err
	}
	g.client = pb.NewGRPCForwarderClient(conn)
	return nil
}

// metricsReporterInterceptor will report metrics from client
func (g *GRPCClient) metricsReporterInterceptor(
	ctx context.Context,
	method string,
	req interface{},
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	l := g.logger.WithFields(log.Fields{
		"method": method,
	})

	startTime := time.Now()

	defer func() {
		timeUsed := float64(time.Since(startTime).Nanoseconds() / int64(time.Millisecond))
		metrics.ClientRequestsResponseTime.WithLabelValues(hostname, method).Observe(timeUsed)
		l.WithFields(log.Fields{
			"timeUsed": timeUsed,
			"reply":    reply.(*pb.Response),
		}).Debug("request processed")
	}()

	err := invoker(ctx, method, req, reply, cc, opts...)
	if err != nil {
		l.WithError(err).Error("error processing request")
		metrics.ClientRequestsFailureCounter.WithLabelValues(hostname, method, err.Error()).Inc()
	} else {
		metrics.ClientRequestsSuccessCounter.WithLabelValues(hostname, method).Inc()
	}
	return err
}

// Wait on pending async send of events
func (g *GRPCClient) Wait() {
	g.wg.Wait()
}
