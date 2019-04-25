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

package testclient

import (
	"time"

	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/client"

	"github.com/sirupsen/logrus"
)

// TestClient is the app structure
type TestClient struct {
	log    logrus.FieldLogger
	config *viper.Viper
	client *client.GRPCClient
}

// NewTestClient creates test client
func NewTestClient(
	log logrus.FieldLogger, config *viper.Viper,
) (*TestClient, error) {
	ct := &TestClient{
		log:    log,
		config: config,
	}
	err := ct.configure()
	return ct, err
}

func (ct *TestClient) loadConfigurationDefaults() {
	ct.config.SetDefault("extensions.kafkaproducer.brokers", "localhost:9192")
}

func (ct *TestClient) configure() error {
	ct.loadConfigurationDefaults()
	c, err := client.NewClient("", ct.config, ct.log, nil)
	if err != nil {
		return err
	}
	ct.client = c
	return nil
}

// Run runs the test client
func (ct *TestClient) Run() {
	if err := ct.client.Send("test-event", map[string]string{
		"some-prop": "some value",
	}); err != nil {
		println(err.Error())
		return
	}
	time.Sleep(1 * time.Second)
	println("done")
}
