package app

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	avro "github.com/topfreegames/avro/eventsgateway/generated"
	"github.com/topfreegames/eventsgateway/forwarder"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
	"google.golang.org/grpc/status"
)

// Server struct
type Server struct {
	log         logrus.FieldLogger
	producer    forwarder.Forwarder
	config      *viper.Viper
	topicPrefix string
}

// NewServer returns a new grpc server
func NewServer(producer forwarder.Forwarder, log logrus.FieldLogger, config *viper.Viper) *Server {
	s := &Server{
		log:      log,
		producer: producer,
		config:   config,
	}
	s.configure()
	return s
}

func (s *Server) configure() {
	s.loadConfigurationDefaults()
	s.topicPrefix = s.config.GetString("extensions.kafkaproducer.topicPrefix")
}

func (s *Server) loadConfigurationDefaults() {
	s.config.SetDefault("extensions.kafkaproducer.topicPrefix", "sv-uploads-")
}

// SendEvent sends a event to kafka
func (s *Server) SendEvent(ctx context.Context, event *pb.Event) (*pb.Response, error) {
	l := s.log.WithField("topic", event.GetTopic())

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

	var buf bytes.Buffer

	l.WithField("event", a).Debugf("serializing event")
	err := a.Serialize(&buf)

	if err != nil {
		l.WithField("event", a).Warnf("error serializing event")
		return nil, status.Errorf(status.Code(err), err.Error())
	}

	topic := fmt.Sprintf("%s%s", s.topicPrefix, event.GetTopic())

	partition, offset, err := s.producer.Produce(topic, buf.Bytes())

	if err != nil {
		s.log.WithError(err).Errorf("error sending event to kafka")
		return nil, status.Errorf(status.Code(err), err.Error())
	}
	s.log.WithFields(logrus.Fields{
		"event":     string(buf.Bytes()),
		"partition": partition,
		"offset":    offset,
	}).Debug("event sent to kafka")

	return nil, nil
}
