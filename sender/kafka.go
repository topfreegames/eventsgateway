package sender

import (
	"bytes"
	"context"
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

type Kafka struct {
	config      *viper.Viper
	logger      logrus.FieldLogger
	producer    forwarder.Forwarder
	topicPrefix string
}

func NewKafka(
	producer forwarder.Forwarder,
	logger logrus.FieldLogger,
	config *viper.Viper,
) Sender {
	k := &Kafka{producer: producer, logger: logger, config: config}
	k.config.SetDefault("extensions.kafkaproducer.topicPrefix", "sv-uploads-")
	k.topicPrefix = k.config.GetString("extensions.kafkaproducer.topicPrefix")
	return k
}

// SendEvents sends a batch of events to kafka
func (k *Kafka) SendEvents(
	ctx context.Context,
	events []*pb.Event,
) []int64 {
	failureIndexes := make([]int64, 0, len(events))
	for i, e := range events {
		if err := k.SendEvent(ctx, e); err != nil {
			failureIndexes = append(failureIndexes, int64(i))
		}
	}
	return failureIndexes
}

// SendEvent sends a event to kafka
func (k *Kafka) SendEvent(
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
	err := a.Serialize(&buf)

	if err != nil {
		l.Warnf("error serializing event")
		return status.Errorf(status.Code(err), err.Error())
	}

	topic := fmt.Sprintf("%s%s", k.topicPrefix, event.GetTopic())

	partition, offset, err := k.producer.Produce(topic, buf.Bytes())
	if err != nil {
		return status.Errorf(status.Code(err), err.Error())
	}
	l.WithFields(logrus.Fields{
		"partition": partition,
		"offset":    offset,
	}).Debug("event sent to kafka")

	return nil
}
