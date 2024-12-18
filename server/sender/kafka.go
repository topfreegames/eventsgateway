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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/spf13/viper"
	avro "github.com/topfreegames/avro/go/eventsgateway/generated"
	"github.com/topfreegames/eventsgateway/v4/server/forwarder"
	"github.com/topfreegames/eventsgateway/v4/server/logger"
	"github.com/topfreegames/eventsgateway/v4/server/metrics"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type KafkaSender struct {
	logger   logger.Logger
	producer forwarder.Forwarder
	config   *viper.Viper
}

func NewKafkaSender(
	producer forwarder.Forwarder,
	logger logger.Logger,
	config *viper.Viper,
) *KafkaSender {
	k := &KafkaSender{producer: producer, logger: logger, config: config}
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
				k.logger.
					WithError(err).
					WithField("topic", events[j].GetTopic()).
					WithField("eventName", events[j].GetName()).
					WithField("eventID", events[j].GetId()).
					Error("failed to send event to kafka")
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
	startTime := time.Now()
	maxMessageBytes := k.config.GetInt("kafka.producer.maxMessageBytes")

	if event.XXX_Size() >= maxMessageBytes {
		err := errors.New(fmt.Sprintf("Event size exceeds kafka.producer.maxMessageBytes %d bytes. Got %d bytes", maxMessageBytes, event.XXX_Size()))
		k.logger.WithError(err).Error("Failed to send event")
		return err
	}

	l := k.logger.WithFields(map[string]interface{}{
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

	topic := event.GetTopic()
	partition, offset, err := k.producer.Produce(ctx, topic, buf.Bytes())

	topicFullName := fmt.Sprintf("%s%s", k.config.GetString("kafka.producer.topicPrefix"), topic)
	kafkaStatus := "ok"
	if err != nil {
		kafkaStatus = "error"
		l.WithError(err).Error("error producing event to kafka")
		metrics.KafkaRequestLatency.WithLabelValues(kafkaStatus, topicFullName).Observe(float64(time.Since(startTime).Milliseconds()))
		return err
	}
	metrics.KafkaRequestLatency.WithLabelValues(kafkaStatus, topicFullName).Observe(float64(time.Since(startTime).Milliseconds()))
	l.WithFields(map[string]interface{}{
		"partition": partition,
		"offset":    offset,
	}).Debug("event sent to kafka")

	return nil
}
