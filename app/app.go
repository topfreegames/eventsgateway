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

package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/topfreegames/eventsgateway/metrics"

	"google.golang.org/grpc"

	"github.com/Shopify/sarama"
	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/forwarder"
	extensions "github.com/topfreegames/extensions/kafka"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"

	"github.com/sirupsen/logrus"
)

var (
	hostname, _ = os.Hostname()
)

// App is the app structure
type App struct {
	host            string
	port            int
	server          *Server
	config          *viper.Viper
	log             logrus.FieldLogger
	eventsForwarder forwarder.Forwarder
}

// NewApp creates a new App object
func NewApp(host string, port int, log logrus.FieldLogger, config *viper.Viper) (*App, error) {
	a := &App{
		host:   host,
		port:   port,
		log:    log,
		config: config,
	}
	err := a.configure()
	return a, err
}

func (a *App) loadConfigurationDefaults() {
	a.config.SetDefault("extensions.kafkaproducer.brokers", "localhost:9192")
	a.config.SetDefault("extensions.kafkaproducer.maxMessageBytes", 3000000)
	a.config.SetDefault("extensions.kafkaproducer.batch.size", 1)
	a.config.SetDefault("extensions.kafkaproducer.linger.ms", 0)
}

func (a *App) configure() error {
	a.loadConfigurationDefaults()
	err := a.configureEventsForwarder()
	if err != nil {
		return err
	}
	return nil
}

func (a *App) configureEventsForwarder() error {
	kafkaConf := sarama.NewConfig()
	kafkaConf.Producer.Return.Errors = true
	kafkaConf.Producer.Return.Successes = true
	kafkaConf.Producer.MaxMessageBytes = a.config.GetInt("extensions.kafkaproducer.maxMessageBytes")
	kafkaConf.Producer.Flush.Bytes = a.config.GetInt("extensions.kafkaproducer.batch.size")
	kafkaConf.Producer.Flush.Frequency = time.Duration(a.config.GetInt("extensions.kafkaproducer.linger.ms")) * time.Millisecond
	kafkaConf.Producer.RequiredAcks = sarama.WaitForLocal
	kafkaConf.Producer.Compression = sarama.CompressionSnappy
	k, err := extensions.NewSyncProducer(a.config, a.log.(*logrus.Logger), kafkaConf)
	if err != nil {
		return err
	}
	sender := NewSender(k, a.log, a.config)
	a.server = NewServer(sender, a.log)
	return nil
}

// metricsReporterInterceptor interceptor
func (a *App) metricsReporterInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {

	l := a.log.WithField("route", info.FullMethod)

	startTime := time.Now()

	defer func() {
		timeUsed := float64(time.Since(startTime).Nanoseconds() / (1000 * 1000))
		metrics.APIResponseTime.WithLabelValues(hostname, info.FullMethod).Observe(timeUsed)
		l.WithField("timeUsed", timeUsed).Debug("request processed")
	}()

	res, err := handler(ctx, req)

	if err != nil {
		l.WithError(err).Error("error processing request")
		metrics.APIRequestsFailureCounter.WithLabelValues(hostname, info.FullMethod, err.Error()).Inc()
	} else {
		metrics.APIRequestsSuccessCounter.WithLabelValues(hostname, info.FullMethod).Inc()
	}

	return res, err
}

// Run runs the app
func (a *App) Run() {
	log := a.log
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", a.host, a.port))
	if err != nil {
		log.Panic(err.Error())
	}
	log.Infof("events gateway listening on %s:%d", a.host, a.port)

	var opts []grpc.ServerOption
	opts = append(opts, grpc.UnaryInterceptor(a.metricsReporterInterceptor))
	grpcServer := grpc.NewServer(opts...)

	pb.RegisterGRPCForwarderServer(grpcServer, a.server)

	grpcServer.Serve(listener)
}
