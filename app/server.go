// MIT License
//
// Copyright (c) 2018 Top Free Games
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
	l := s.log.WithFields(logrus.Fields{
		"topic": event.GetTopic(),
		"event": event,
	})

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

	l.Debugf("serializing event")
	err := a.Serialize(&buf)

	if err != nil {
		l.Warnf("error serializing event")
		return nil, status.Errorf(status.Code(err), err.Error())
	}

	topic := fmt.Sprintf("%s%s", s.topicPrefix, event.GetTopic())

	partition, offset, err := s.producer.Produce(topic, buf.Bytes())

	if err != nil {
		s.log.WithError(err).Errorf("error sending event to kafka")
		return nil, status.Errorf(status.Code(err), err.Error())
	}
	s.log.WithFields(logrus.Fields{
		"partition": partition,
		"offset":    offset,
	}).Debug("event sent to kafka")

	return &pb.Response{}, nil
}
