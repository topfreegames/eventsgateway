// eventsgateway
// +build integration
// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2019 Top Free Games <backend@tfgco.com>

package client_test

import (
	"github.com/Shopify/sarama"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/viper"
	wrapper "github.com/topfreegames/eventsgateway/v4/logger"
	logruswrapper "github.com/topfreegames/eventsgateway/v4/logger/logrus"
	"time"

	"testing"

	. "github.com/topfreegames/eventsgateway/v4/testing"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Suite")
}

func expectOneMessage(messageID string, messages chan *sarama.ConsumerMessage, errors chan *sarama.ConsumerError) *sarama.ConsumerMessage {
	select {
	case msg := <- messages:
		Expect(string(msg.Value)).To(ContainSubstring(messageID))
		return msg
	case err := <- errors:
		Expect(err).NotTo(HaveOccurred())
	case <-time.NewTimer(1 * time.Second).C:
		Fail("timed out waiting for message")
	}
	return nil
}

var (
	config *viper.Viper
	logger        *logrus.Logger
	wrappedLogger wrapper.Logger
)

var _ = BeforeEach(func() {
	logger, _ = test.NewNullLogger()
	logger.Level = logrus.DebugLevel
	// logger.Out = os.Stdout // uncomment this to view logs
	wrappedLogger = logruswrapper.NewWithLogger(logger)
	config, _ = GetDefaultConfig()
})
