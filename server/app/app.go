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
	"github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	goMetrics "github.com/rcrowley/go-metrics"
	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/v4/server/forwarder"
	"github.com/topfreegames/eventsgateway/v4/server/logger"
	"github.com/topfreegames/eventsgateway/v4/server/metrics"
	"github.com/topfreegames/eventsgateway/v4/server/sender"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
	jaegerclient "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"io"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
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

	err := a.configureEventsForwarder()
	if err != nil {
		return err
	}
	return nil
}

func (a *App) configureOpenTracing() (io.Closer, error) {
	// InitTracer initializes a tracer using the Config instance's parameters

	jcfg := jaegercfg.Configuration{
		ServiceName: a.config.GetString("tracing.serviceName"),
		Sampler: &jaegercfg.SamplerConfig{
			Type:  a.config.GetString("tracing.jaeger.samplerType"),
			Param: a.config.GetFloat64("tracing.jaeger.samplerParam"),
		},
		Reporter: &jaegercfg.ReporterConfig{
			LocalAgentHostPort: fmt.Sprintf("%s:%d", a.config.GetString("tracing.jaeger.host"), a.config.GetInt("tracing.jaeger.port")),
			LogSpans:           a.config.GetBool("tracing.logSpans"),
		},
		Disabled: !a.config.GetBool("tracing.enabled"),
	}

	tracer, closer, err := jcfg.NewTracer(
		jaegercfg.Logger(jaegerclient.StdLogger),
		jaegercfg.MaxTagValueLength(a.config.GetInt("tracing.maxTagValueLength")),
		jaegercfg.Tag("environment", a.config.GetString("tracing.environment")))
	if err != nil {
		return nil, err
	}
	opentracing.SetGlobalTracer(tracer)
	return closer, nil
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
	events := []*pb.Event{}
	retry := "0"
	payloadSize := 0
	switch t := req.(type) {
	case *pb.Event:
		event := req.(*pb.Event)
		events = append(events, event)
		payloadSize = proto.Size(event)
	case *pb.SendEventsRequest:
		request := req.(*pb.SendEventsRequest)
		events = append(events, request.Events...)
		retry = fmt.Sprintf("%d", request.Retry)
		payloadSize = proto.Size(request)
	default:
		a.log.WithField("route", info.FullMethod).Infof("Unexpected request type %T", t)
	}

	topic := events[0].Topic
	l := a.log.
		WithField("route", info.FullMethod).
		WithField("topic", topic)

	metrics.APIPayloadSize.WithLabelValues(
		info.FullMethod,
		topic,
	).Observe(float64(payloadSize))

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

	metrics.StartServer(a.config)
	var opts []grpc.ServerOption

	if a.config.GetBool("tracing.enabled") {
		tracerCloser, err := a.configureOpenTracing()
		if err != nil {
			log.Panic(err.Error())
		}
		defer func() {
			err := tracerCloser.Close()
			log.Info("Closing SERVER tracer ------------------------------------------------------")
			if err != nil {
				log.Error(err.Error())
			}
		}()

	}

	opts = append(
		opts,
		grpc.UnaryInterceptor(a.metricsReporterInterceptor),
		grpc.ChainUnaryInterceptor(
			a.metricsReporterInterceptor,
			otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer()),
		),
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
