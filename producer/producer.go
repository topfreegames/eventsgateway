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

package producer

import (
	"context"
	"github.com/topfreegames/eventsgateway/v4/client"
	"time"

	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/v4/logger"
)

// Producer is the app strupure
type Producer struct {
	log    logger.Logger
	config *viper.Viper
	client *client.Client
}

// NewProducer creates test client
func NewProducer(
	log logger.Logger, config *viper.Viper,
) (*Producer, error) {
	p := &Producer{
		log:    log,
		config: config,
	}
	err := p.configure()
	return p, err
}

func (p *Producer) configure() error {
	c, err := client.New("", p.config, p.log, nil)
	if err != nil {
		return err
	}
	p.client = c
	return nil
}

// Run runs the test client
func (p *Producer) Run() {
	if err := p.client.Send(context.Background(), "test-event", map[string]string{
		"some-prop": "some value",
	}); err != nil {
		println(err.Error())
		return
	}
	time.Sleep(1 * time.Second)
	println("done")
}
