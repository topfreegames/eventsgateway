// eventsgateway
// +build integration
// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2019 Top Free Games <backend@tfgco.com>

package client_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/viper"

	"testing"

	. "github.com/topfreegames/eventsgateway/testing"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Suite")
}

var (
	config *viper.Viper
	logger *logrus.Logger
)

var _ = BeforeEach(func() {
	logger, _ = test.NewNullLogger()
	logger.Level = logrus.DebugLevel
	config, _ = GetDefaultConfig()
})
