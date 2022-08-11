// eventsgateway
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
	"github.com/topfreegames/eventsgateway/app"
	"github.com/topfreegames/eventsgateway/client"
	"github.com/topfreegames/eventsgateway/testing"
	"google.golang.org/grpc"
)

var _ = Describe("Sync Client", func() {
	var (
		a        *app.App
		c        *client.Client
		kafkaTopic string
		consumer *testing.Consumer
	)
	name := "EventName"
	props := map[string]string{
		"prop1": "val1",
		"prop2": "val2",
	}

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
		if a != nil {
			a.Stop()
		}
	})

	startAppAndClient := func() {
		var err error
		a, err = app.NewApp("0.0.0.0", 5000, wrappedLogger, config)
		Expect(err).NotTo(HaveOccurred())
		go a.Run()
		config.Set("client.async", false)
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
		It("Should send event synchronously", func() {
			startAppAndClient()

			props["messageID"] = uuid.New().String()
			err := c.Send(context.Background(), name, props)
			Expect(err).NotTo(HaveOccurred())
			err = c.GracefulStop()
			Expect(err).NotTo(HaveOccurred())

			msgs, errs := consumer.Consume(fmt.Sprintf("sv-uploads-%s", kafkaTopic))
			expectOneMessage(props["messageID"], msgs, errs)
		})

		It("Should send multiple events synchronously", func() {
			startAppAndClient()
			const msgAmount = 3
			var err error
			var msgIDs []string

			for i := 0; i < msgAmount; i++ {
				msgID := uuid.NewString()
				msgIDs = append(msgIDs, msgID)

				props["messageID"] = msgID
				err = c.Send(context.Background(), name, props)
				Expect(err).NotTo(HaveOccurred())
			}

			msgs, errs := consumer.Consume(fmt.Sprintf("sv-uploads-%s", kafkaTopic))
			for _, msgID := range msgIDs {
				expectOneMessage(msgID, msgs, errs)
			}
		})
	})
})
