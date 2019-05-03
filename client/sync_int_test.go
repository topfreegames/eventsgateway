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
	"time"

	"github.com/topfreegames/eventsgateway/app"
	"github.com/topfreegames/eventsgateway/client"
	extensions "github.com/topfreegames/extensions/kafka"
	"google.golang.org/grpc"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sync Client", func() {
	var (
		a        *app.App
		c        *client.Client
		consumer *extensions.Consumer
	)
	name := "EventName"
	props := map[string]string{
		"prop1": "val1",
		"prop2": "val2",
	}

	BeforeEach(func() {
		var err error
		consumer, err = extensions.NewConsumer(config, logger)
		Expect(err).NotTo(HaveOccurred())
		go consumer.ConsumeLoop()
		time.Sleep(1 * time.Second) // wait consumer receive assign partition
	})

	AfterEach(func() {
		consumer.Cleanup()
		if a != nil {
			a.Stop()
		}
	})

	startAppAndClient := func() {
		var err error
		a, err = app.NewApp("0.0.0.0", 5000, logger, config)
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
			err := c.Send(context.Background(), name, props)
			Expect(err).NotTo(HaveOccurred())
			err = c.GracefulStop()
			Expect(err).NotTo(HaveOccurred())
			<-*consumer.MessagesChannel()
		})

		It("Should send multiple events synchronously", func() {
			startAppAndClient()
			var err error
			err = c.Send(context.Background(), name, props)
			Expect(err).NotTo(HaveOccurred())
			err = c.Send(context.Background(), name, props)
			Expect(err).NotTo(HaveOccurred())
			err = c.Send(context.Background(), name, props)
			Expect(err).NotTo(HaveOccurred())
			err = c.GracefulStop()
			Expect(err).NotTo(HaveOccurred())
			<-*consumer.MessagesChannel()
			<-*consumer.MessagesChannel()
			<-*consumer.MessagesChannel()
		})
	})
})
