//go:build unit
// +build unit

package app_test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/spf13/viper"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	avro "github.com/topfreegames/avro/go/eventsgateway/generated"
	"github.com/topfreegames/eventsgateway/v4/server/app"
	"github.com/topfreegames/eventsgateway/v4/server/sender"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
)

func initConfig() *viper.Viper {
	config = viper.New()
	config.SetConfigFile("../config/test.yaml")
	config.SetConfigType("yaml")
	config.SetEnvPrefix("eventsgateway")
	config.AddConfigPath(".")
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AutomaticEnv()

	if err := config.ReadInConfig(); err != nil {
		fmt.Printf("Error loading config file: %s\n", config.ConfigFileUsed())
	}
	return config
}

var _ = Describe("Client", func() {
	var (
		s     *app.Server
		nowMs int64
	)

	BeforeEach(func() {
		nowMs = time.Now().UnixNano() / 1000000
		sender := sender.NewKafkaSender(mockForwarder, log, initConfig())
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
		It("should fail send exceeds message size", func() {
			msg := []string{"a", "a"}
			for _ = range 30000 {
				msg = append(msg, "a")
			}

			ctx := context.Background()
			e := &pb.Event{
				Id:    "someid",
				Name:  "someName",
				Topic: "sv-uploads-sometopic",
				Props: map[string]string{
					"bigmessage": strings.Join(msg, ""),
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
				})

			res, err := s.SendEvent(ctx, e)
			Expect(res).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Event size exceeds kafka.producer.maxMessageBytes 30000 bytes. Got 30069 bytes"))
		})
	})
})
