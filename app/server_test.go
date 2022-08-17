// +build unit

package app_test

import (
	"bytes"
	"context"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	avro "github.com/topfreegames/avro/go/eventsgateway/generated"
	"github.com/topfreegames/eventsgateway/app"
	"github.com/topfreegames/eventsgateway/sender"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
)

var _ = Describe("Client", func() {
	var (
		s     *app.Server
		nowMs int64
	)

	BeforeEach(func() {
		nowMs = time.Now().UnixNano() / 1000000
		sender := sender.NewKafkaSender(mockForwarder, log)
		s = app.NewServer(sender, log)
		Expect(s).NotTo(BeNil())
	})

	Describe("SendEvent Tests", func() {
		It("should fail if topic is not set", func() {
			ctx := context.Background()
			e := &pb.Event{
				Id:        "someId",
				Name:      "someEvent",
				Props:     map[string]string{},
				Timestamp: nowMs,
			}
			res, err := s.SendEvent(ctx, e)
			Expect(res).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("rpc error: code = FailedPrecondition desc = id, topic, name and timestamp should be set"))
		})

		It("should fail if name is not set", func() {
			ctx := context.Background()
			e := &pb.Event{
				Id:        "someId",
				Name:      "",
				Topic:     "sometopic",
				Props:     map[string]string{},
				Timestamp: nowMs,
			}
			res, err := s.SendEvent(ctx, e)
			Expect(res).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("rpc error: code = FailedPrecondition desc = id, topic, name and timestamp should be set"))
		})

		It("should fail if id is not set", func() {
			ctx := context.Background()
			e := &pb.Event{
				Id:        "",
				Name:      "someName",
				Topic:     "sometopic",
				Props:     map[string]string{},
				Timestamp: nowMs,
			}
			res, err := s.SendEvent(ctx, e)
			Expect(res).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("rpc error: code = FailedPrecondition desc = id, topic, name and timestamp should be set"))
		})

		It("should fail if timestamp is not set", func() {
			ctx := context.Background()
			e := &pb.Event{
				Id:    "someId",
				Name:  "someName",
				Topic: "sometopic",
				Props: map[string]string{},
			}
			res, err := s.SendEvent(ctx, e)
			Expect(res).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("rpc error: code = FailedPrecondition desc = id, topic, name and timestamp should be set"))
		})

		It("should send event", func() {
			ctx := context.Background()
			e := &pb.Event{
				Id:        "someid",
				Name:      "someName",
				Topic:     "sv-uploads-sometopic",
				Props:     map[string]string{},
				Timestamp: nowMs,
			}

			mockForwarder.EXPECT().Produce(gomock.Eq("sv-uploads-sometopic"), gomock.Any()).Do(
				func(topic string, aevent []byte) {
					r := bytes.NewReader(aevent)
					ev, err := avro.DeserializeEvent(r)
					Expect(err).NotTo(HaveOccurred())
					Expect(ev.Id).To(Equal(e.GetId()))
					Expect(ev.Name).To(Equal(e.GetName()))
					Expect(ev.ClientTimestamp).To(Equal(e.GetTimestamp()))
					Expect(ev.ServerTimestamp).To(BeNumerically("~", nowMs, 10))
				})

			res, err := s.SendEvent(ctx, e)
			Expect(res).NotTo(BeNil())
			Expect(err).NotTo(HaveOccurred())
		})

		It("should send event with props", func() {
			ctx := context.Background()
			e := &pb.Event{
				Id:    "someid",
				Name:  "someName",
				Topic: "sv-uploads-sometopic",
				Props: map[string]string{
					"test1": "lalala",
					"test2": "bla",
				},
				Timestamp: nowMs,
			}

			mockForwarder.EXPECT().Produce(gomock.Eq("sv-uploads-sometopic"), gomock.Any()).Do(
				func(topic string, aevent []byte) {
					r := bytes.NewReader(aevent)
					ev, err := avro.DeserializeEvent(r)
					Expect(err).NotTo(HaveOccurred())
					Expect(ev.Id).To(Equal(e.GetId()))
					Expect(ev.Name).To(Equal(e.GetName()))
					Expect(ev.ClientTimestamp).To(Equal(e.GetTimestamp()))
					Expect(ev.ServerTimestamp).To(BeNumerically("~", nowMs, 10))
					Expect(ev.Props).To(BeEquivalentTo(map[string]string{
						"test1": "lalala",
						"test2": "bla",
					}))
				})

			res, err := s.SendEvent(ctx, e)
			Expect(res).NotTo(BeNil())
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
