package app

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/topfreegames/eventsgateway/forwarder"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
)

// Server struct
type Server struct {
	log      logrus.FieldLogger
	producer forwarder.Forwarder
}

// NewServer returns a new grpc server
func NewServer(producer forwarder.Forwarder, log logrus.FieldLogger) *Server {
	return &Server{
		log:      log,
		producer: producer,
	}
}

// SendEvent sends a event to kafka
func (s *Server) SendEvent(ctx context.Context, event *pb.Event) (*pb.Response, error) {
	s.log.Debugf("received event with id: %s, name: %s, topic: %s, props: %s",
		event.GetId(),
		event.GetName(),
		event.GetTopic(),
		event.GetProps(),
	)

	return &pb.Response{
		Message: "ack",
		Code:    200,
	}, nil
}
