// eventsgateway
//go:build unit
// +build unit

// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2019 Top Free Games <backend@tfgco.com>

package client

import (
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	logruswrapper "github.com/topfreegames/eventsgateway/v4/logger/logrus"
	t "github.com/topfreegames/eventsgateway/v4/testing"
	mockpb "github.com/topfreegames/protos/eventsgateway/grpc/mock"
)

var _ = Describe("Client Whitebox", func() {
	var (
		c *Client
	)

	BeforeEach(func() {
		var err error
		logger, _ := test.NewNullLogger()
		logger.Level = logrus.DebugLevel
		config, _ := t.GetDefaultConfig()

		mockCtrl := gomock.NewController(GinkgoT())
		mockGRPCClient := mockpb.NewMockGRPCForwarderClient(mockCtrl)
		config.Set("client.async", true)
		c, err = New(
			"",
			config,
			logruswrapper.NewWithLogger(logger),
			mockGRPCClient,
		)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("NewClient", func() {
		It("Should have gRPCClientAsync as grpc client", func() {
			_, ok := c.GetGRPCClient().(*gRPCClientAsync)
			Expect(ok).To(BeTrue())
		})
	})
})
