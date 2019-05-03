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

package loadtestclient

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/client"

	"github.com/sirupsen/logrus"
)

// LoadTestClient is the app strultcure
type LoadTestClient struct {
	log                logrus.FieldLogger
	config             *viper.Viper
	client             *client.Client
	propsSize          string
	duration           time.Duration
	randSleepCeilingMs int
}

// NewLoadTestClient creates test client
func NewLoadTestClient(
	log logrus.FieldLogger, config *viper.Viper,
) (*LoadTestClient, error) {
	rand.Seed(time.Now().Unix())
	ltc := &LoadTestClient{
		log:    log,
		config: config,
	}
	err := ltc.configure()
	return ltc, err
}

func (ltc *LoadTestClient) configure() error {
	ltc.config.Set("client.kafkatopic", randomTopic())
	c, err := client.NewClient("", ltc.config, ltc.log, nil)
	if err != nil {
		return err
	}
	ltc.client = c
	ltc.config.SetDefault("loadtestclient.duration", "10s")
	ltc.duration = ltc.config.GetDuration("loadtestclient.duration")
	ltc.config.SetDefault("loadtestclient.randSleepCeilingMs", 500)
	ltc.randSleepCeilingMs = ltc.config.GetInt("loadtestclient.randSleepCeilingMs")
	ltc.config.SetDefault("loadtestclient.propsSize", "small")
	ltc.propsSize = ltc.config.GetString("loadtestclient.propsSize")
	return nil
}

// Run runs the load test client
func (ltc *LoadTestClient) Run() {
	ticker := time.NewTicker(ltc.duration)
	defer ticker.Stop()
	ctx := context.Background()
	sentCounter := 0
	for {
		select {
		default:
			props := buildProps(ltc.propsSize)
			time.Sleep(time.Duration(rand.Intn(ltc.randSleepCeilingMs)) * time.Millisecond)
			if rand.Intn(2) == 0 {
				ltc.client.Send(ctx, "load test event", props)
			} else {
				ltc.client.SendToTopic(ctx, "load test event", props, randomTopic())
			}
			sentCounter++
		case <-ticker.C:
			fmt.Printf("Sent %d events in %s\n", sentCounter, ltc.duration)
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			<-c
			return
		}
	}
}

func randomTopic() string {
	return []string{
		"clemente",
		"sussie",
		"fay",
		"mallie",
		"vern",
		"kramer",
		"costanza",
	}[rand.Intn(7)]
}

func buildProps(size string) map[string]string {
	m := map[string]int{
		"small":  5,
		"medium": 9,
		"large":  13,
		"jumbo":  23,
	}
	n := rand.Intn(m[size])
	props := map[string]string{}
	for i := 0; i < n; i++ {
		props[uuid.NewV4().String()] = uuid.NewV4().String()
	}
	return props
}
