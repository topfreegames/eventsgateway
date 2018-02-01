// eventsgateway
// +build unit
// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2018 Top Free Games <backend@tfgco.com>

package client_test

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/viper"

	"testing"

	. "github.com/topfreegames/eventsgateway/testing"
	mockpb "github.com/topfreegames/protos/eventsgateway/grpc/mock"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Suite")
}

var (
	config         *viper.Viper
	hook           *test.Hook
	logger         *logrus.Logger
	mockCtrl       *gomock.Controller
	mockGRPCClient *mockpb.MockGRPCForwarderClient
)

var _ = BeforeEach(func() {
	logger, hook = test.NewNullLogger()
	logger.Level = logrus.DebugLevel
	config, _ = GetDefaultConfig()

	mockCtrl = gomock.NewController(GinkgoT())
	mockGRPCClient = mockpb.NewMockGRPCForwarderClient(mockCtrl)
})
