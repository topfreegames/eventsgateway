//go:build unit

package sender

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	"github.com/topfreegames/eventsgateway/v4/server/forwarder"
	"github.com/topfreegames/eventsgateway/v4/server/logger"
)

const e2eTopic = "sv-uploads-topic"

func newFailingKafka(t *testing.T, partitions int32, failing ...int32) *sarama.MockBroker {
	t.Helper()
	broker := sarama.NewMockBroker(t, 1)
	metadata := sarama.NewMockMetadataResponse(t).SetBroker(broker.Addr(), broker.BrokerID())
	for p := int32(0); p < partitions; p++ {
		metadata = metadata.SetLeader(e2eTopic, p, broker.BrokerID())
	}
	produce := sarama.NewMockProduceResponse(t)
	for _, p := range failing {
		produce = produce.SetError(e2eTopic, p, sarama.ErrNotEnoughReplicas)
	}
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"ApiVersionsRequest": sarama.NewMockApiVersionsResponse(t),
		"MetadataRequest":    metadata,
		"ProduceRequest":     produce,
	})
	return broker
}

func sendThroughKafka(t *testing.T, partitioner string, failing ...int32) []int64 {
	t.Helper()
	broker := newFailingKafka(t, 4, failing...)
	defer broker.Close()
	cfg := testConfig()
	cfg.Set("kafka.producer.brokers", broker.Addr())
	cfg.Set("kafka.producer.partitioner", partitioner)
	cfg.Set("kafka.producer.net.maxOpenRequests", 10)
	cfg.Set("kafka.producer.net.dialTimeout", "500ms")
	cfg.Set("kafka.producer.net.readTimeout", "250ms")
	cfg.Set("kafka.producer.net.writeTimeout", "250ms")
	cfg.Set("kafka.producer.timeout", "250ms")
	cfg.Set("kafka.producer.linger.ms", 1)
	cfg.Set("kafka.logger.enabled", false)
	fwd, err := forwarder.NewKafkaForwarder(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer fwd.Close()
	return NewKafkaSender(fwd, &logger.NullLogger{}, cfg).SendEvents(context.Background(), testEvents(50))
}

func TestSendEventsThroughKafkaDeliversEveryEvent(t *testing.T) {
	for _, partitioner := range []string{"hash", "sticky"} {
		if failures := sendThroughKafka(t, partitioner); len(failures) != 0 {
			t.Errorf("%s: expected no failures, got %v", partitioner, failures)
		}
	}
}

func TestSendEventsThroughKafkaReportsEveryEventOfAFailingPartition(t *testing.T) {
	failures := sendThroughKafka(t, "sticky", 0, 1, 2, 3)
	if len(failures) != 50 {
		t.Fatalf("expected all 50 events reported as failed, got %d: %v", len(failures), failures)
	}
	for i, f := range failures {
		if f != int64(i) {
			t.Fatalf("expected failure indexes 0 to 49 in order, got %v", failures)
		}
	}
}

func TestSendEventsThroughKafkaReportsOnlyEventsOfTheFailingPartition(t *testing.T) {
	failures := sendThroughKafka(t, "hash", 2)
	if len(failures) == 0 || len(failures) == 50 {
		t.Fatalf("expected some but not all events to fail, got %d", len(failures))
	}
}
