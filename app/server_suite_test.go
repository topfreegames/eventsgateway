// +build unit
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

package app_test

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"

	"testing"

	"github.com/topfreegames/eventsgateway/logger"
	"github.com/topfreegames/eventsgateway/mocks"
	. "github.com/topfreegames/eventsgateway/testing"
	mockpb "github.com/topfreegames/protos/eventsgateway/grpc/mock"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App suite")
}

var (
	config         *viper.Viper
	log            logger.Logger
	mockCtrl       *gomock.Controller
	mockGRPCServer *mockpb.MockGRPCForwarderServer
	mockForwarder  *mocks.MockForwarder
)

var _ = BeforeEach(func() {
	log = &logger.NullLogger{}
	config, _ = GetDefaultConfig()

	mockCtrl = gomock.NewController(GinkgoT())
	mockGRPCServer = mockpb.NewMockGRPCForwarderServer(mockCtrl)
	mockForwarder = mocks.NewMockForwarder(mockCtrl)
})
