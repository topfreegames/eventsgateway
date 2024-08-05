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
	"fmt"
	"github.com/opentracing/opentracing-go"
	jaegerclient "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/v4/logger"
)

// LoadTest holds runners clients
type LoadTest struct {
	config       *viper.Viper
	duration     time.Duration
	log          logger.Logger
	threads      int
	runners      []*runner
	wg           sync.WaitGroup
	tracerCloser io.Closer
}

func (lt *LoadTest) configureOpenTracing(appName string) (io.Closer, error) {
	// InitTracer initializes a tracer using the Config instance's parameters
	lt.log.Infof("Configuring Opentracing %s.", appName)

	jcfg := jaegercfg.Configuration{
		ServiceName: fmt.Sprintf("%s-%s", lt.config.GetString("tracing.serviceNamePrefix"), appName),
		Sampler: &jaegercfg.SamplerConfig{
			Type:  lt.config.GetString("tracing.jaeger.samplerType"),
			Param: lt.config.GetFloat64("tracing.jaeger.samplerParam"),
		},
		Reporter: &jaegercfg.ReporterConfig{
			LocalAgentHostPort: fmt.Sprintf("%s:%d", lt.config.GetString("tracing.jaeger.host"), lt.config.GetInt("tracing.jaeger.port")),
			LogSpans:           lt.config.GetBool("tracing.logSpans"),
		},
		Disabled: !lt.config.GetBool("tracing.enabled"),
	}

	tracer, closer, err := jcfg.NewTracer(
		jaegercfg.Logger(jaegerclient.StdLogger),
		jaegercfg.MaxTagValueLength(lt.config.GetInt("tracing.maxTagValueLength")),
		jaegercfg.Tag("environment", lt.config.GetString("tracing.environment")))
	if err != nil {
		return nil, err
	}
	opentracing.SetGlobalTracer(tracer)
	return closer, nil
}

// NewLoadTest ctor
func NewLoadTest(log logger.Logger, config *viper.Viper) (*LoadTest, error) {
	rand.Seed(time.Now().Unix())
	lt := &LoadTest{
		log:    log,
		config: config,
	}
	lt.config.SetDefault("loadtestclient.duration", "10s")
	lt.duration = lt.config.GetDuration("loadtestclient.duration")
	lt.config.SetDefault("loadtestclient.threads", 2)
	lt.threads = lt.config.GetInt("loadtestclient.threads")
	lt.runners = make([]*runner, lt.threads)

	tracerCloser, err := lt.configureOpenTracing("loadtest")

	lt.tracerCloser = tracerCloser
	if err != nil {
		lt.log.Panic(err.Error())
		panic(err)
	}
	for i := 0; i < lt.threads; i++ {
		runner, err := newRunner(log, config)
		if err != nil {
			return nil, err
		}
		lt.runners[i] = runner
	}
	return lt, nil
}

// Run starts all runners
func (lt *LoadTest) Run() {
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
	_ = lt.tracerCloser.Close()
}
