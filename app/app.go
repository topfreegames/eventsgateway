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
	"github.com/topfreegames/eventsgateway/forwarder"
	"go.opentelemetry.io/otel/propagation"
	"net"
	"time"

	goMetrics "github.com/rcrowley/go-metrics"
	"github.com/topfreegames/eventsgateway/logger"
	"github.com/topfreegames/eventsgateway/metrics"
	"github.com/topfreegames/eventsgateway/sender"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/spf13/viper"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
)

const (
	OTLServiceName = "events-gateway"
)

// App is the app structure
type App struct {
	Server     *Server // tests manipulate this field
	config     *viper.Viper
	grpcServer *grpc.Server
	host       string
	log        logger.Logger
	port       int
}

// NewApp creates a new App object
func NewApp(host string, port int, log logger.Logger, config *viper.Viper) (*App, error) {
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
	a.config.SetDefault("otlp.enabled", false)
	a.config.SetDefault("kafka.producer.net.maxOpenRequests", 10)
	a.config.SetDefault("kafka.producer.net.dialTimeout", "500ms")
	a.config.SetDefault("kafka.producer.net.readTimeout", "250ms")
	a.config.SetDefault("kafka.producer.net.writeTimeout", "250ms")
	a.config.SetDefault("kafka.producer.net.keepAlive", "60s")
	a.config.SetDefault("kafka.producer.brokers", "localhost:9192")
	a.config.SetDefault("kafka.producer.maxMessageBytes", 1000000)
	a.config.SetDefault("kafka.producer.timeout", "250ms")
	a.config.SetDefault("kafka.producer.batch.size", 1000000)
	a.config.SetDefault("kafka.producer.linger.ms", 1)
	a.config.SetDefault("kafka.producer.retry.max", 0)
	a.config.SetDefault("kafka.producer.clientId", "eventsgateway")
	a.config.SetDefault("kafka.producer.topicPrefix", "sv-uploads-")
	a.config.SetDefault("server.maxConnectionIdle", "20s")
	a.config.SetDefault("server.maxConnectionAge", "20s")
	a.config.SetDefault("server.maxConnectionAgeGrace", "5s")
	a.config.SetDefault("server.Time", "10s")
	a.config.SetDefault("server.Timeout", "500ms")
}

func (a *App) configure() error {
	a.loadConfigurationDefaults()

	if a.config.GetBool("otlp.enabled") {
		if err := a.configureOTel(); err != nil {
			return err
		}
	}

	err := a.configureEventsForwarder()
	if err != nil {
		return err
	}
	return nil
}

func (a *App) configureOTel() error {

	traceExporter, err := otlptracegrpc.New(context.Background())

	if err != nil {
		a.log.Error("Unable to create a OTL exporter", err)
		return err
	}

	traceResources := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(OTLServiceName),
	)

	traceProvider := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(traceExporter),
		tracesdk.WithResource(traceResources),
	)
	otel.SetTracerProvider(traceProvider)

	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(propagator)

	return nil
}

func (a *App) configureEventsForwarder() error {
	goMetrics.UseNilMetrics = true
	k, err := forwarder.NewKafkaForwarder(a.config)
	if err != nil {
		return err
	}
	kafkaSender := sender.NewKafkaSender(k, a.log)
	a.Server = NewServer(kafkaSender, a.log)
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
				"error processing request",
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

	otelPropagator := otelgrpc.WithPropagators(otel.GetTextMapPropagator())
	otelTracerProvider := otelgrpc.WithTracerProvider(otel.GetTracerProvider())

	opts = append(opts, grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		a.metricsReporterInterceptor,
		otelgrpc.UnaryServerInterceptor(otelTracerProvider, otelPropagator),
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
