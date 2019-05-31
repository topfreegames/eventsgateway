// eventsgateway
// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2019 Top Free Games <backend@tfgco.com>

package sender

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	avro "github.com/topfreegames/avro/go/eventsgateway/generated"
	"github.com/topfreegames/eventsgateway/forwarder"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type KafkaSender struct {
	config      *viper.Viper
	logger      logrus.FieldLogger
	producer    forwarder.Forwarder
	topicPrefix string
}

func NewKafkaSender(
	producer forwarder.Forwarder,
	logger logrus.FieldLogger,
	config *viper.Viper,
) *KafkaSender {
	k := &KafkaSender{producer: producer, logger: logger, config: config}
	k.config.SetDefault("extensions.kafkaproducer.topicPrefix", "sv-uploads-")
	k.topicPrefix = k.config.GetString("extensions.kafkaproducer.topicPrefix")
	return k
}

// SendEvents sends a batch of events to kafka
func (k *KafkaSender) SendEvents(
	ctx context.Context,
	events []*pb.Event,
) []int64 {
	wg := sync.WaitGroup{}
	wg.Add(len(events))
	failureIndexes := make([]int64, 0, len(events))
	for i := range events {
		j := i
		go func() {
			if err := k.SendEvent(ctx, events[j]); err != nil {
				failureIndexes = append(failureIndexes, int64(j))
			}
			wg.Done()
		}()
	}
	wg.Wait()
	return failureIndexes
}

// SendEvent sends a event to kafka
func (k *KafkaSender) SendEvent(
	ctx context.Context,
	event *pb.Event,
) error {
	l := k.logger.WithFields(logrus.Fields{
		"topic": event.GetTopic(),
		"event": event,
	})

	if event.GetId() == "" ||
		event.GetTopic() == "" ||
		event.GetName() == "" ||
		event.GetTimestamp() == int64(0) {
		return status.Errorf(codes.FailedPrecondition, "id, topic, name and timestamp should be set")
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

	a.ServerTimestamp = time.Now().UnixNano() / 1000000
	a.ClientTimestamp = event.GetTimestamp()

	var buf bytes.Buffer

	l.Debugf("serializing event")
	if err := a.Serialize(&buf); err != nil {
		l.Warnf("error serializing event")
		return err
	}

	topic := fmt.Sprintf("%s%s", k.topicPrefix, event.GetTopic())

	partition, offset, err := k.producer.Produce(topic, buf.Bytes())
	if err != nil {
		l.WithError(err).Error("failed to send event to kafka")
		return err
	}
	l.WithFields(logrus.Fields{
		"partition": partition,
		"offset":    offset,
	}).Debug("event sent to kafka")

	return nil
}
