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
	"log"
	"net"
	"os"
	"strings"
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

	"github.com/Shopify/sarama"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/spf13/viper"
	kafka "github.com/topfreegames/go-extensions-kafka"
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
	a.config.SetDefault("jaeger.url", "http://localhost:14267/api/traces")
	a.config.SetDefault("jaeger.disabled", true)
	a.config.SetDefault("jaeger.samplingProbability", 0.1)
	a.config.SetDefault("jaeger.serviceName", "events-gateway")
	a.config.SetDefault("extensions.kafkaproducer.net.maxOpenRequests", 10)
	a.config.SetDefault("extensions.kafkaproducer.net.dialTimeout", "500ms")
	a.config.SetDefault("extensions.kafkaproducer.net.readTimeout", "250ms")
	a.config.SetDefault("extensions.kafkaproducer.net.writeTimeout", "250ms")
	a.config.SetDefault("extensions.kafkaproducer.net.keepAlive", "60s")
	a.config.SetDefault("extensions.kafkaproducer.brokers", "localhost:9192")
	a.config.SetDefault("extensions.kafkaproducer.maxMessageBytes", 1000000)
	a.config.SetDefault("extensions.kafkaproducer.timeout", "250ms")
	a.config.SetDefault("extensions.kafkaproducer.batch.size", 1000000)
	a.config.SetDefault("extensions.kafkaproducer.linger.ms", 1)
	a.config.SetDefault("extensions.kafkaproducer.retry.max", 0)
	a.config.SetDefault("extensions.kafkaproducer.clientId", "eventsgateway")
	a.config.SetDefault("server.maxConnectionIdle", "20s")
	a.config.SetDefault("server.maxConnectionAge", "20s")
	a.config.SetDefault("server.maxConnectionAgeGrace", "5s")
	a.config.SetDefault("server.Time", "10s")
	a.config.SetDefault("server.Timeout", "500ms")
}

func (a *App) configure() error {
	a.loadConfigurationDefaults()
	if err := a.configureOTEL(context.Background()); err != nil {
		return err
	}
	err := a.configureEventsForwarder()
	if err != nil {
		return err
	}
	return nil
}

func (a *App) configureOTEL(ctx context.Context) error {

	traceExporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithInsecure())

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

	return nil
}

func (a *App) configureEventsForwarder() error {
	goMetrics.UseNilMetrics = true
	if a.config.GetBool("extensions.sarama.logger.enabled") {
		sarama.Logger = log.New(os.Stdout, "sarama", log.Llongfile)
	}
	kafkaConf := sarama.NewConfig()
	kafkaConf.Net.MaxOpenRequests = a.config.GetInt("extensions.kafkaproducer.net.maxOpenRequests")
	kafkaConf.Net.DialTimeout = a.config.GetDuration("extensions.kafkaproducer.net.dialTimeout")
	kafkaConf.Net.ReadTimeout = a.config.GetDuration("extensions.kafkaproducer.net.readTimeout")
	kafkaConf.Net.WriteTimeout = a.config.GetDuration("extensions.kafkaproducer.net.writeTimeout")
	kafkaConf.Net.KeepAlive = a.config.GetDuration("extensions.kafkaproducer.net.keepAlive")
	kafkaConf.Producer.Return.Errors = true
	kafkaConf.Producer.Return.Successes = true
	kafkaConf.Producer.MaxMessageBytes = a.config.GetInt("extensions.kafkaproducer.maxMessageBytes")
	kafkaConf.Producer.Timeout = a.config.GetDuration("extensions.kafkaproducer.timeout")
	kafkaConf.Producer.Flush.Bytes = a.config.GetInt("extensions.kafkaproducer.batch.size")
	kafkaConf.Producer.Flush.Frequency = time.Duration(a.config.GetInt("extensions.kafkaproducer.linger.ms")) * time.Millisecond
	kafkaConf.Producer.Retry.Max = a.config.GetInt("extensions.kafkaproducer.retry.max")
	kafkaConf.Producer.RequiredAcks = sarama.WaitForLocal
	kafkaConf.Producer.Compression = sarama.CompressionSnappy
	kafkaConf.ClientID = a.config.GetString("extensions.kafkaproducer.clientId")
	kafkaConf.Version = sarama.V2_1_0_0
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

	otelOpts := otelgrpc.WithTracerProvider(otel.GetTracerProvider())

	opts = append(opts, grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		a.metricsReporterInterceptor,
		otelgrpc.UnaryServerInterceptor(otelOpts),
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
