// eventsgateway
// +build unit
// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2017 Top Free Games <backend@tfgco.com>

package client_test

import (
	"context"
	"errors"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/topfreegames/eventsgateway/client"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sync Client", func() {
	var (
		c   *client.Client
		now int64
	)
	name := "EventName"
	props := map[string]string{
		"prop1": "val1",
		"prop2": "val2",
	}

	BeforeEach(func() {
		var err error
		c, err = client.NewClient("", config, log, mockGRPCClient)
		Expect(err).NotTo(HaveOccurred())
		now = time.Now().UnixNano() / 1000000
	})

	Describe("Send", func() {
		var (
			c   *client.Client
			now int64
		)
		name := "EventName"
		props := map[string]string{
			"prop1": "val1",
			"prop2": "val2",
		}

		BeforeEach(func() {
			var err error
			c, err = client.NewClient("", config, log, mockGRPCClient)
			Expect(err).NotTo(HaveOccurred())
			now = time.Now().UnixNano() / 1000000
		})

		It("should send event to configured topic", func() {
			mockGRPCClient.EXPECT().SendEvent(
				gomock.Any(),
				gomock.Any(),
			).Do(func(ctx context.Context, event *pb.Event) {
				Expect(event.Id).NotTo(BeEmpty())
				Expect(event.Name).To(Equal(name))
				Expect(event.Topic).To(Equal("test-topic"))
				Expect(event.Props).To(Equal(props))
				Expect(event.Timestamp).To(BeNumerically("~", now, 100))
			}).Return(nil, nil)

			err := c.Send(context.Background(), name, props)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail if event forward fails", func() {
			mockGRPCClient.EXPECT().SendEvent(
				gomock.Any(),
				gomock.Any(),
			).Return(nil, errors.New("olar"))

			err := c.Send(context.Background(), name, props)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("olar"))
		})
	})

	Describe("SendToTopic", func() {
		It("should send event to specific topic", func() {
			topic := "custom-topic"
			mockGRPCClient.EXPECT().SendEvent(
				gomock.Any(),
				gomock.Any(),
			).Do(func(ctx context.Context, event *pb.Event) {
				Expect(event.Id).NotTo(BeEmpty())
				Expect(event.Name).To(Equal(name))
				Expect(event.Topic).To(Equal(topic))
				Expect(event.Props).To(Equal(props))
				Expect(event.Timestamp).To(BeNumerically("~", now, 100))
			}).Return(nil, nil)

			err := c.SendToTopic(context.Background(), name, props, topic)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail if event forward fails", func() {
			mockGRPCClient.EXPECT().SendEvent(
				gomock.Any(),
				gomock.Any(),
			).Return(nil, errors.New("olar"))

			err := c.Send(context.Background(), name, props)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("olar"))
		})
	})
})
