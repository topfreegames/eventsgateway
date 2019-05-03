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
	"github.com/topfreegames/eventsgateway/mocks"
	extensions "github.com/topfreegames/extensions/kafka"
	"google.golang.org/grpc"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Async Client", func() {
	var (
		a *app.App
		c *client.Client
		s *mocks.MockSender
	)
	name := "EventName"
	props := map[string]string{
		"prop1": "val1",
		"prop2": "val2",
	}

	AfterEach(func() {
		if a != nil {
			a.Stop()
		}
	})

	Describe("Not mocking producer", func() {
		var (
			consumer *extensions.Consumer
		)

		BeforeEach(func() {
			var err error
			consumer, err = extensions.NewConsumer(config, logger)
			Expect(err).NotTo(HaveOccurred())
			go consumer.ConsumeLoop()
			consumer.WaitUntilReady()
		})

		AfterEach(func() {
			consumer.Cleanup()
		})

		startAppAndClient := func() {
			var err error
			a, err = app.NewApp("0.0.0.0", 5000, logger, config)
			Expect(err).NotTo(HaveOccurred())
			go a.Run()
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
				startAppAndClient()
				err := c.Send(context.Background(), name, props)
				Expect(err).NotTo(HaveOccurred())
				err = c.GracefulStop()
				Expect(err).NotTo(HaveOccurred())
				select {
				case msg := <-*consumer.MessagesChannel():
					Expect(string(msg)).To(ContainSubstring(name))
					// assert on the message?
				case <-time.NewTimer(1 * time.Second).C:
					Fail("timed out waiting for message")
				}
			})
		})
	})

	Describe("Mocking producer", func() {
		startAppAndClient := func() {
			var err error
			a, err = app.NewApp("0.0.0.0", 5000, logger, config)
			Expect(err).NotTo(HaveOccurred())
			s = mocks.NewMockSender()
			server := app.NewServer(s, logger)
			a.Server = server
			go a.Run()
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
				startAppAndClient()
				err := c.Send(context.Background(), name, props)
				Expect(err).NotTo(HaveOccurred())
				err = c.GracefulStop()
				Expect(err).NotTo(HaveOccurred())
				Expect(s.GetTotalSent()).To(Equal(1))
			})

			It("Should send multiple events asynchronously", func() {
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
				Expect(s.GetTotalSent()).To(Equal(3))
			})

			Describe("Flush configurations", func() {
				It("Should flush at every flushSize = 1 events", func() {
					config.Set("client.flushInterval", 1*time.Second)
					config.Set("client.flushSize", 1)
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
					Expect(s.GetTotalSent()).To(Equal(3))
					Expect(s.GetTotalCalls()).To(Equal(3))
				})

				It("Should flush at every flushSize = 3 events", func() {
					config.Set("client.flushInterval", 1*time.Second)
					config.Set("client.flushSize", 3)
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
					Expect(s.GetTotalSent()).To(Equal(3))
					Expect(s.GetTotalCalls()).To(Equal(1))
				})

				It("Should flush at every flushInterval = 5ms", func() {
					config.Set("client.flushInterval", 5*time.Millisecond)
					config.Set("client.flushSize", 3)
					startAppAndClient()
					var err error
					err = c.Send(context.Background(), name, props)
					Expect(err).NotTo(HaveOccurred())
					err = c.Send(context.Background(), name, props)
					Expect(err).NotTo(HaveOccurred())
					time.Sleep(10 * time.Millisecond)
					err = c.Send(context.Background(), name, props)
					Expect(err).NotTo(HaveOccurred())
					err = c.GracefulStop()
					Expect(err).NotTo(HaveOccurred())
					Expect(s.GetTotalSent()).To(Equal(3))
					Expect(s.GetTotalCalls()).To(Equal(2))
				})
			})

			Describe("Retries", func() {
				It("Should retry only failed indexes", func() {
					config.Set("client.maxRetries", 3)
					config.Set("client.retryInterval", 1*time.Nanosecond)
					config.Set("client.flushInterval", 1*time.Second)
					config.Set("client.flushSize", 5)
					startAppAndClient()
					s.SetFailureIndexesOrder([][]int64{
						[]int64{0, 3},
					})
					var err error
					err = c.Send(context.Background(), name, props)
					Expect(err).NotTo(HaveOccurred())
					err = c.Send(context.Background(), name, props)
					Expect(err).NotTo(HaveOccurred())
					err = c.Send(context.Background(), name, props)
					Expect(err).NotTo(HaveOccurred())
					err = c.Send(context.Background(), name, props)
					Expect(err).NotTo(HaveOccurred())
					err = c.Send(context.Background(), name, props)
					Expect(err).NotTo(HaveOccurred())
					err = c.GracefulStop()
					Expect(err).NotTo(HaveOccurred())
					Expect(s.GetTotalSent()).To(Equal(5))
					Expect(s.GetTotalCalls()).To(Equal(2))
					events := s.GetFirstCallEvents()
					freqs := s.GetEventsIdsFreqs()
					Expect(freqs[events[0].Id]).To(Equal(2))
					Expect(freqs[events[1].Id]).To(Equal(1))
					Expect(freqs[events[2].Id]).To(Equal(1))
					Expect(freqs[events[3].Id]).To(Equal(2))
					Expect(freqs[events[4].Id]).To(Equal(1))
				})

				It("Should stop after maxRetries = 0", func() {
					config.Set("client.maxRetries", 0)
					config.Set("client.retryInterval", 1*time.Nanosecond)
					config.Set("client.flushInterval", 1*time.Second)
					config.Set("client.flushSize", 5)
					startAppAndClient()
					s.SetFailureIndexesOrder([][]int64{
						[]int64{0, 3},
					})
					var err error
					err = c.Send(context.Background(), name, props)
					Expect(err).NotTo(HaveOccurred())
					err = c.Send(context.Background(), name, props)
					Expect(err).NotTo(HaveOccurred())
					err = c.Send(context.Background(), name, props)
					Expect(err).NotTo(HaveOccurred())
					err = c.Send(context.Background(), name, props)
					Expect(err).NotTo(HaveOccurred())
					err = c.Send(context.Background(), name, props)
					Expect(err).NotTo(HaveOccurred())
					err = c.GracefulStop()
					Expect(err).NotTo(HaveOccurred())
					Expect(s.GetTotalSent()).To(Equal(3))
					Expect(s.GetTotalCalls()).To(Equal(1))
					events := s.GetFirstCallEvents()
					freqs := s.GetEventsIdsFreqs()
					Expect(freqs[events[0].Id]).To(Equal(1))
					Expect(freqs[events[1].Id]).To(Equal(1))
					Expect(freqs[events[2].Id]).To(Equal(1))
					Expect(freqs[events[3].Id]).To(Equal(1))
					Expect(freqs[events[4].Id]).To(Equal(1))
				})
			})
		})
	})
})
