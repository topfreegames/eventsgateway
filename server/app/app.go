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
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/topfreegames/eventsgateway/v4/server/forwarder"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	jaegerPropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"

	goMetrics "github.com/rcrowley/go-metrics"
	"github.com/topfreegames/eventsgateway/v4/server/logger"
	"github.com/topfreegames/eventsgateway/v4/server/metrics"
	"github.com/topfreegames/eventsgateway/v4/server/sender"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/spf13/viper"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
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
	a.config.SetDefault("kafka.producer.idempotent", false)
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
	a.config.SetDefault("prometheus.enabled", "true") // always true on the API side
	a.config.SetDefault("prometheus.port", ":9091")
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
	traceExporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%d", a.config.GetString("otlp.jaegerHost"), a.config.GetInt("otlp.jaegerPort"))),
		otlptracegrpc.WithInsecure())

	if err != nil {
		a.log.Error("Unable to create a OTL exporter", err)
		return err
	}

	traceProvider := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(traceExporter),
		// Record information about this application in a Resource.
		tracesdk.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName(a.config.GetString("otlp.serviceName")),
				attribute.String("environment", a.config.GetString("server.environment")),
			),
		),
		tracesdk.WithSampler(
			tracesdk.ParentBased(
				tracesdk.TraceIDRatioBased(
					a.config.GetFloat64("otlp.traceSamplingRatio")),
				tracesdk.WithRemoteParentSampled(tracesdk.AlwaysSample()),
				tracesdk.WithRemoteParentNotSampled(tracesdk.NeverSample()),
				tracesdk.WithLocalParentSampled(tracesdk.AlwaysSample()),
				tracesdk.WithLocalParentNotSampled(tracesdk.NeverSample()))),
	)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			jaegerPropagator.Jaeger{},
			propagation.TraceContext{},
			propagation.Baggage{}),
	)
	otel.SetTracerProvider(traceProvider)

	return nil
}

func (a *App) configureEventsForwarder() error {
	goMetrics.UseNilMetrics = true
	k, err := forwarder.NewKafkaForwarder(a.config)
	if err != nil {
		return err
	}
	kafkaSender := sender.NewKafkaSender(k, a.log, a.config)
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
	events := []*pb.Event{}
	payloadSize := 0
	switch t := req.(type) {
	case *pb.Event:
		event := req.(*pb.Event)
		events = append(events, event)
		payloadSize = proto.Size(event)
	case *pb.SendEventsRequest:
		request := req.(*pb.SendEventsRequest)
		events = append(events, request.Events...)
		payloadSize = proto.Size(request)
	default:
		a.log.WithField("route", info.FullMethod).Infof("Unexpected request type %T", t)
	}

	topic := events[0].Topic
	metrics.APIPayloadSize.WithLabelValues(
		topic).Observe(float64(payloadSize))

	startTime := time.Now()
	res, err := handler(ctx, req)
	responseStatus := "ok"
	if err != nil {
		responseStatus = "error"
		metrics.APIResponseTime.WithLabelValues(
			info.FullMethod,
			responseStatus,
			topic,
		).Observe(float64(time.Since(startTime).Milliseconds()))
		a.log.
			WithField("route", info.FullMethod).
			WithField("topic", topic).
			WithError(err).Error("error processing request")
		return res, err
	}
	metrics.APIResponseTime.WithLabelValues(
		info.FullMethod,
		responseStatus,
		topic,
	).Observe(float64(time.Since(startTime).Milliseconds()))
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

	metrics.StartServer(a.config)
	var opts []grpc.ServerOption

	otelPropagator := otelgrpc.WithPropagators(otel.GetTextMapPropagator())
	otelTracerProvider := otelgrpc.WithTracerProvider(otel.GetTracerProvider())

	opts = append(
		opts,
		grpc.UnaryInterceptor(a.metricsReporterInterceptor),
		grpc.StatsHandler(otelgrpc.NewServerHandler(otelPropagator, otelTracerProvider)),
		grpc.KeepaliveParams(keepalive.ServerParameters{
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
