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

package loadtest

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/sirupsen/logrus"
)

// LoadTest holds runners clients
type LoadTest struct {
	config   *viper.Viper
	duration time.Duration
	log      logrus.FieldLogger
	threads  int
	runners  []*runner
	wg       sync.WaitGroup
}

// NewLoadTest ctor
func NewLoadTest(log logrus.FieldLogger, config *viper.Viper) (*LoadTest, error) {
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
}
