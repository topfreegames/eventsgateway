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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/v4/logger"

	"testing"

	. "github.com/topfreegames/eventsgateway/v4/testing"
	mockpb "github.com/topfreegames/protos/eventsgateway/grpc/mock"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Suite")
}

var (
	config         *viper.Viper
	log            logger.Logger
	mockCtrl       *gomock.Controller
	mockGRPCClient *mockpb.MockGRPCForwarderClient
)

var _ = BeforeEach(func() {
	log = &logger.NullLogger{}
	config, _ = GetDefaultConfig()

	mockCtrl = gomock.NewController(GinkgoT())
	mockGRPCClient = mockpb.NewMockGRPCForwarderClient(mockCtrl)
})
