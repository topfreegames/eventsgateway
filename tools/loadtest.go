// MIT License
//
// Copyright (c) 2019 Top Free Games
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

package tools

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	jaegerclient "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/v4/logger"
)

// LoadTest holds runners clients
type LoadTest struct {
	config   *viper.Viper
	duration time.Duration
	log      logger.Logger
	threads  int
	runners  []*runner
	wg       sync.WaitGroup
}

// NewLoadTest ctor
func NewLoadTest(log logger.Logger, config *viper.Viper) (*LoadTest, error) {
	rand.Seed(time.Now().Unix())
	lt := &LoadTest{
		log:    log,
		config: config,
	}
	lt.config.SetDefault("loadtestclient.duration", "10s")
	lt.config.SetDefault("loadtestclient.threads", 2)

	lt.duration = lt.config.GetDuration("loadtestclient.duration")
	lt.threads = lt.config.GetInt("loadtestclient.threads")

	var err error
	if lt.config.GetBool("loadtestclient.opentelemetry.enabled") {
		err = lt.configureOpenTelemetry()
	} else if lt.config.GetBool("loadtestclient.opentracing.enabled") {
		_, err = lt.configureOpenTracing()
	}

	if err != nil {
		return nil, err
	}

	lt.runners = make([]*runner, lt.threads)
	for i := 0; i < lt.threads; i++ {
		runner, err := newRunner(log, config)
		if err != nil {
			return nil, err
		}
		lt.runners[i] = runner
	}
	return lt, nil
}

func (lt *LoadTest) configureOpenTracing() (io.Closer, error) {
	jcfg := jaegercfg.Configuration{
		ServiceName: lt.config.GetString("loadtestclient.opentracing.serviceName"),
		Sampler: &jaegercfg.SamplerConfig{
			Type:  lt.config.GetString("loadtestclient.opentracing.samplerType"),
			Param: lt.config.GetFloat64("loadtestclient.opentracing.samplerParam"),
		},
		Reporter: &jaegercfg.ReporterConfig{
			LocalAgentHostPort: fmt.Sprintf("%s:%d", lt.config.GetString("loadtestclient.opentracing.jaegerHost"), lt.config.GetInt("loadtestclient.opentracing.jaegerPort")),
			LogSpans:           lt.config.GetBool("loadtestclient.opentracing.logSpans"),
		},
		Disabled: !lt.config.GetBool("loadtestclient.opentracing.enabled"),
	}

	tracer, closer, err := jcfg.NewTracer(
		jaegercfg.Logger(jaegerclient.StdLogger),
		jaegercfg.MaxTagValueLength(2500),
		jaegercfg.Tag("environment", "development"))
	if err != nil {
		return nil, err
	}
	opentracing.SetGlobalTracer(tracer)
	return closer, nil
}

func (lt *LoadTest) configureOpenTelemetry() error {
	traceExporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%d", lt.config.GetString("loadtestclient.opentelemetry.jaegerHost"), lt.config.GetInt("loadtestclient.opentelemetry.jaegerPort"))),
		otlptracegrpc.WithInsecure())

	if err != nil {
		lt.log.Error("Unable to create a OTL exporter", err)
		return err
	}

	traceProvider := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(traceExporter),
		// Record information about this application in a Resource.
		tracesdk.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(lt.config.GetString("loadtestclient.opentelemetry.serviceName")),
			),
		),
		tracesdk.WithSampler(
			tracesdk.ParentBased(
				tracesdk.TraceIDRatioBased(
					lt.config.GetFloat64("loadtestclient.opentelemetry.traceSamplingRatio")),
				tracesdk.WithRemoteParentSampled(tracesdk.AlwaysSample()),
				tracesdk.WithRemoteParentNotSampled(tracesdk.NeverSample()),
				tracesdk.WithLocalParentSampled(tracesdk.AlwaysSample()),
				tracesdk.WithLocalParentNotSampled(tracesdk.NeverSample())),
		),
	)
	otel.SetTracerProvider(traceProvider)

	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{})
	otel.SetTextMapPropagator(propagator)

	return nil
}

func (lt *LoadTest) startMetricsServer() {
	go func() {
		envEnabled := lt.config.GetString("loadtestclient.prometheus.enabled")
		if envEnabled != "true" {
			log.Warn("Prometheus web server disabled")
			return
		}

		r := mux.NewRouter()
		r.Handle("/metrics", promhttp.Handler())

		s := &http.Server{
			Addr:           lt.config.GetString("loadtestclient.prometheus.port"),
			ReadTimeout:    8 * time.Second,
			WriteTimeout:   8 * time.Second,
			MaxHeaderBytes: 1 << 20,
			Handler:        r,
		}
		log.Info("Starting Metrics server...")
		log.Fatal(s.ListenAndServe())
	}()
}

// Run starts all runners
func (lt *LoadTest) Run() {
	lt.startMetricsServer()
	lt.wg.Add(lt.threads)
	for _, runner := range lt.runners {
		go runner.run()
	}
	lt.wg.Wait()
	sentCounter := uint64(0)
	for _, runner := range lt.runners {
		sentCounter += runner.sentCounter
	}
	fmt.Printf("Sent %d events in %s\n", sentCounter, lt.duration)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
