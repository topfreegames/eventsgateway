package app

import (
	"bytes"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	avro "github.com/topfreegames/avro/go/eventsgateway/generated"
	"github.com/topfreegames/eventsgateway/forwarder"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Sender interface {
	SendEvents(msg *pb.Events) (*pb.Response, error)
	SendEvent(msg *pb.Event) (*pb.Response, error)
}

type sender struct {
	config      *viper.Viper
	logger      logrus.FieldLogger
	producer    forwarder.Forwarder
	topicPrefix string
}

func NewSender(
	producer forwarder.Forwarder,
	logger logrus.FieldLogger,
	config *viper.Viper,
) Sender {
	s := &sender{producer: producer, logger: logger, config: config}
	s.configure()
	return s
}

func (s *sender) configure() {
	s.loadConfigurationDefaults()
	s.topicPrefix = s.config.GetString("extensions.kafkaproducer.topicPrefix")
}

func (s *sender) loadConfigurationDefaults() {
	s.config.SetDefault("extensions.kafkaproducer.topicPrefix", "sv-uploads-")
}

// SendEvents sends a batch of events to kafka
func (s *sender) SendEvents(msg *pb.Events) (*pb.Response, error) {
	for i := range msg.Events {
		if _, err := s.SendEvent(msg.Events[i]); err != nil {
			return &pb.Response{FailureIndex: int64(i)}, err
		}
	}
	return &pb.Response{}, nil
}

// SendEvent sends a event to kafka
func (s *sender) SendEvent(event *pb.Event) (*pb.Response, error) {
	l := s.logger.WithFields(logrus.Fields{
		"topic": event.GetTopic(),
		"event": event,
	})

	if event.GetId() == "" ||
		event.GetTopic() == "" ||
		event.GetName() == "" ||
		event.GetTimestamp() == int64(0) {
		return nil, status.Errorf(codes.FailedPrecondition, "id, topic, name and timestamp should be set")
	}

	l.Debugf("received event with id: %s, name: %s, topic: %s, props: %s",
		event.GetId(),
		event.GetName(),
		event.GetTopic(),
		event.GetProps(),
	)
	// serialize
	a := avro.NewEvent()
	a.Id = event.GetId()
	a.Name = event.GetName()
	a.Props = event.GetProps()

	a.ServerTimestamp = time.Now().UnixNano() / int64(time.Millisecond)
	a.ClientTimestamp = event.GetTimestamp()

	var buf bytes.Buffer

	l.Debugf("serializing event")
	err := a.Serialize(&buf)

	if err != nil {
		l.Warnf("error serializing event")
		return nil, status.Errorf(status.Code(err), err.Error())
	}

	topic := fmt.Sprintf("%s%s", s.topicPrefix, event.GetTopic())

	partition, offset, err := s.producer.Produce(topic, buf.Bytes())

	if err != nil {
		return nil, status.Errorf(status.Code(err), err.Error())
	}
	l.WithFields(logrus.Fields{
		"partition": partition,
		"offset":    offset,
	}).Debug("event sent to kafka")

	return &pb.Response{}, nil
}
