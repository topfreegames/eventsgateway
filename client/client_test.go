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
	"github.com/topfreegames/eventsgateway/v4/client"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
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
		c, err = client.New("", config, log, mockGRPCClient)
		Expect(err).NotTo(HaveOccurred())
		now = time.Now().UnixNano() / 1000000
	})

	Describe("NewClient", func() {
		It("should return client if no error", func() {
			c, err := client.New("", config, log, mockGRPCClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
		})

		It("should return an error if no kafka topic", func() {
			config.Set("client.kafkatopic", "")
			c, err := client.New("", config, log, mockGRPCClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no kafka topic informed"))
			Expect(c).To(BeNil())
		})

		It("should return an error if no server address", func() {
			config.Set("client.grpc.serveraddress", "")
			c, err := client.New("", config, log, mockGRPCClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no grpc server address informed"))
			Expect(c).To(BeNil())
		})
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
			c, err = client.New("", config, log, mockGRPCClient)
			Expect(err).NotTo(HaveOccurred())
			now = time.Now().UnixNano() / int64(time.Millisecond)
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
			topic := "custom-topic"
			mockGRPCClient.EXPECT().SendEvent(
				gomock.Any(),
				gomock.Any(),
			).Return(nil, errors.New("olar"))

			err := c.SendToTopic(context.Background(), name, props, topic)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("olar"))
		})
	})

	Describe("SendAtTime", func() {
		It("should send event with a specific timestamp", func() {
			t1 := time.Now()
			expectedT1 := t1.UnixNano() / 1000000

			mockGRPCClient.EXPECT().SendEvent(
				gomock.Any(),
				gomock.Any(),
			).Do(func(ctx context.Context, event *pb.Event) {
				Expect(event.Id).NotTo(BeEmpty())
				Expect(event.Name).To(Equal(name))
				Expect(event.Topic).To(Equal("test-topic"))
				Expect(event.Props).To(Equal(props))
				Expect(event.Timestamp).To(BeNumerically("~", expectedT1, 100))
			}).Return(nil, nil)

			err := c.SendAtTime(context.Background(), name, props, t1)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail if event forward fails", func() {
			t1 := time.Now()

			mockGRPCClient.EXPECT().SendEvent(
				gomock.Any(),
				gomock.Any(),
			).Return(nil, errors.New("olar"))

			err := c.SendAtTime(context.Background(), name, props, t1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("olar"))
		})
	})
})
