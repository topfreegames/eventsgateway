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
	"strings"
	"time"

	"github.com/topfreegames/eventsgateway/metrics"
	"github.com/topfreegames/eventsgateway/sender"
	"github.com/topfreegames/extensions/jaeger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/Shopify/sarama"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/spf13/viper"
	kafka "github.com/topfreegames/go-extensions-kafka"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"

	"github.com/sirupsen/logrus"
)

// App is the app structure
type App struct {
	Server     *Server // tests manipulate this field
	config     *viper.Viper
	grpcServer *grpc.Server
	host       string
	log        logrus.FieldLogger
	port       int
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
	a.config.SetDefault("jaeger.disabled", true)
	a.config.SetDefault("jaeger.samplingProbability", 0.1)
	a.config.SetDefault("jaeger.serviceName", "events-gateway")
	a.config.SetDefault("extensions.kafkaproducer.net.maxOpenRequests", 100)
	a.config.SetDefault("extensions.kafkaproducer.brokers", "localhost:9192")
	a.config.SetDefault("extensions.kafkaproducer.maxMessageBytes", 1000000)
	a.config.SetDefault("extensions.kafkaproducer.timeout", "250ms")
	a.config.SetDefault("extensions.kafkaproducer.batch.size", 1000000)
	a.config.SetDefault("extensions.kafkaproducer.linger.ms", 0)
	a.config.SetDefault("extensions.kafkaproducer.retry.max", 0)
	a.config.SetDefault("server.maxConnectionIdle", "20s")
	a.config.SetDefault("server.maxConnectionAge", "20s")
	a.config.SetDefault("server.maxConnectionAgeGrace", "5s")
	a.config.SetDefault("server.Time", "10s")
	a.config.SetDefault("server.Timeout", "500ms")
}

func (a *App) configure() error {
	a.loadConfigurationDefaults()
	if err := a.configureJaeger(); err != nil {
		return err
	}
	err := a.configureEventsForwarder()
	if err != nil {
		return err
	}
	return nil
}

func (a *App) configureJaeger() error {
	opts := jaeger.Options{
		Disabled:    a.config.GetBool("jaeger.disabled"),
		Probability: a.config.GetFloat64("jaeger.samplingProbability"),
		ServiceName: a.config.GetString("jaeger.serviceName"),
	}
	_, err := jaeger.Configure(opts)
	return err
}

func (a *App) configureEventsForwarder() error {
	kafkaConf := sarama.NewConfig()
	kafkaConf.Net.MaxOpenRequests = a.config.GetInt("extensions.kafkaproducer.net.maxOpenRequests")
	kafkaConf.Producer.Return.Errors = true
	kafkaConf.Producer.Return.Successes = true
	kafkaConf.Producer.MaxMessageBytes = a.config.GetInt("extensions.kafkaproducer.maxMessageBytes")
	kafkaConf.Producer.Timeout = a.config.GetDuration("extensions.kafkaproducer.timeout")
	kafkaConf.Producer.Flush.Bytes = a.config.GetInt("extensions.kafkaproducer.batch.size")
	kafkaConf.Producer.Flush.Frequency = time.Duration(a.config.GetInt("extensions.kafkaproducer.linger.ms")) * time.Millisecond
	kafkaConf.Producer.Retry.Max = a.config.GetInt("extensions.kafkaproducer.retry.max")
	kafkaConf.Producer.RequiredAcks = sarama.WaitForLocal
	kafkaConf.Producer.Compression = sarama.CompressionSnappy
	brokers := strings.Split(a.config.GetString("extensions.kafkaproducer.brokers"), ",")
	k, err := kafka.NewSyncProducer(brokers, kafkaConf)
	if err != nil {
		return err
	}
	sender := sender.NewKafkaSender(k, a.log, a.config)
	a.Server = NewServer(sender, a.log)
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

	events := []*pb.Event{}
	retry := "0"
	switch t := req.(type) {
	case *pb.Event:
		events = append(events, req.(*pb.Event))
	case *pb.SendEventsRequest:
		events = append(events, req.(*pb.SendEventsRequest).Events...)
		retry = fmt.Sprintf("%d", req.(*pb.SendEventsRequest).Retry)
	default:
		l.Infof("Unexpected request type %T", t)
	}

	defer func(startTime time.Time) {
		elapsedTime := float64(time.Since(startTime).Nanoseconds() / (1000 * 1000))
		for _, e := range events {
			metrics.APIResponseTime.WithLabelValues(
				info.FullMethod,
				e.Topic,
				retry,
			).Observe(elapsedTime)
		}
		l.WithField("elapsedTime", elapsedTime).Debug("request processed")
	}(time.Now())

	reportedFailures := false
	res, err := handler(ctx, req)
	if err != nil {
		l.WithError(err).Error("error processing request")
		for _, e := range events {
			metrics.APIRequestsFailureCounter.WithLabelValues(
				info.FullMethod,
				e.Topic,
				retry,
				err.Error(),
			).Inc()
		}
		reportedFailures = true
		return res, err
	}
	failureIndexes := []int64{}
	if _, ok := res.(*pb.SendEventsResponse); ok {
		failureIndexes = res.(*pb.SendEventsResponse).FailureIndexes
	}
	fC := 0
	for i, e := range events {
		if !reportedFailures && len(failureIndexes) > fC && int64(i) == failureIndexes[fC] {
			metrics.APIRequestsFailureCounter.WithLabelValues(
				info.FullMethod,
				e.Topic,
				retry,
				"couldn't produce event",
			).Inc()
			fC++
		}
		metrics.APIRequestsSuccessCounter.WithLabelValues(
			info.FullMethod,
			e.Topic,
			retry,
		).Inc()
	}
	return res, nil
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
	tracer := opentracing.GlobalTracer()
	opts = append(opts, grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		a.metricsReporterInterceptor,
		otgrpc.OpenTracingServerInterceptor(tracer),
	)))
	opts = append(opts, grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle:     a.config.GetDuration("server.maxConnectionIdle"),
		MaxConnectionAge:      a.config.GetDuration("server.maxConnectionAge"),
		MaxConnectionAgeGrace: a.config.GetDuration("server.maxConnectionAgeGrace"),
		Time:                  a.config.GetDuration("server.Time"),
		Timeout:               a.config.GetDuration("server.Timeout"),
	}))
	a.grpcServer = grpc.NewServer(opts...)

	pb.RegisterGRPCForwarderServer(a.grpcServer, a.Server)
	if err := a.grpcServer.Serve(listener); err != nil {
		log.Panic(err.Error())
	}
}

func (a *App) Stop() {
	a.grpcServer.GracefulStop()
}
