// eventsgateway
//go:build integration
// +build integration

// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2019 Top Free Games <backend@tfgco.com>

package client_test

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/topfreegames/eventsgateway/v4/client"
	"github.com/topfreegames/eventsgateway/v4/testing"
	"google.golang.org/grpc"
)

var _ = Describe("Async Client", func() {
	var (
		c          *client.Client
		kafkaTopic string
	)
	name := "EventName"
	props := map[string]string{
		"prop1": "val1",
		"prop2": "val2",
	}

	Describe("Not mocking producer", func() {
		var (
			consumer *testing.Consumer
		)

		BeforeEach(func() {
			var err error
			consumer, err = testing.NewConsumer(config.GetStringSlice("kafka.producer.brokers")[0])
			Expect(err).NotTo(HaveOccurred())

			kafkaTopic = fmt.Sprintf("test-%s", uuid.New().String())
			config.Set("client.kafkatopic", kafkaTopic)
		})

		AfterEach(func() {
			err := consumer.Clean()
			Expect(err).NotTo(HaveOccurred())
		})

		initClient := func() {
			var err error
			config.Set("client.async", true)
			c, err = client.NewClient(
				"",
				config,
				logger,
				nil,
				grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
			)
			Expect(err).NotTo(HaveOccurred())
		}

		Describe("Send", func() {
			It("Should send event asynchronously", func() {
				initClient()
				props["messageID"] = uuid.New().String()
				err := c.Send(context.Background(), name, props)
				Expect(err).NotTo(HaveOccurred())
				err = c.GracefulStop()
				Expect(err).NotTo(HaveOccurred())

				msgs, errs := consumer.Consume(fmt.Sprintf("sv-uploads-%s", kafkaTopic))
				_ = expectOneMessage(props["messageID"], msgs, errs)
			})
		})
	})

})
