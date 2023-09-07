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
	"context"
	"math/rand"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/v4/client"
	"github.com/topfreegames/eventsgateway/v4/logger"
)

type runner struct {
	client             *client.Client
	config             *viper.Viper
	duration           time.Duration
	log                logger.Logger
	randPropsSize      string
	randSleepCeilingMs int
	sentCounter        uint64
	threads            int
}

func newRunner(
	log logger.Logger, config *viper.Viper,
) (*runner, error) {
	rand.Seed(time.Now().Unix())
	r := &runner{
		log:    log,
		config: config,
	}
	err := r.configure()
	return r, err
}

func (r *runner) configure() error {
	r.config.Set("client.kafkatopic", randomTopic())
	c, err := client.New("", r.config, r.log, nil)
	if err != nil {
		return err
	}
	r.client = c
	r.config.SetDefault("loadtestclient.duration", "10s")
	r.duration = r.config.GetDuration("loadtestclient.duration")
	r.config.SetDefault("loadtestclient.randSleepCeilingMs", 500)
	r.randSleepCeilingMs = r.config.GetInt("loadtestclient.randSleepCeilingMs")
	r.config.SetDefault("loadtestclient.randPropsSize", "small")
	r.randPropsSize = r.config.GetString("loadtestclient.randPropsSize")
	return nil
}

func (r *runner) run() {
	ctx := context.Background()
	stop := time.After(r.duration)
	for {
		select {
		default:
			props := buildProps(r.randPropsSize)
			time.Sleep(time.Duration(rand.Intn(r.randSleepCeilingMs)) * time.Millisecond)
			if rand.Intn(2) == 0 {
				r.client.Send(ctx, "load test event", props)
			} else {
				r.client.SendToTopic(ctx, "load test event", props, randomTopic())
			}
			r.sentCounter++
		case <-stop:
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
		"small":  11,
		"medium": 17,
		"large":  29,
		"jumbo":  37,
	}
	n := rand.Intn(m[size])
	if n < m[size] {
		n = m[size]
	}
	props := map[string]string{}
	for i := 0; i < n; i++ {
		props[uuid.NewV4().String()] = uuid.NewV4().String()
	}
	return props
}
