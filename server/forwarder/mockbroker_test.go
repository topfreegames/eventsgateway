//go:build unit

package forwarder

import (
	"context"
	"sync"
	"testing"

	"github.com/IBM/sarama"
)

func newMockKafka(t *testing.T, topic string, partitions int32) *sarama.MockBroker {
	t.Helper()
	broker := sarama.NewMockBroker(t, 1)
	metadata := sarama.NewMockMetadataResponse(t).SetBroker(broker.Addr(), broker.BrokerID())
	for p := int32(0); p < partitions; p++ {
		metadata = metadata.SetLeader(topic, p, broker.BrokerID())
	}
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"ApiVersionsRequest": sarama.NewMockApiVersionsResponse(t),
		"MetadataRequest":    metadata,
		"ProduceRequest":     sarama.NewMockProduceResponse(t),
	})
	return broker
}

func producedPartitions(t *testing.T, partitioner string, messages int) map[int32]int {
	t.Helper()
	broker := newMockKafka(t, "sv-uploads-topic", 4)
	defer broker.Close()
	cfg := appDefaults()
	cfg.Set("kafka.producer.brokers", broker.Addr())
	cfg.Set("kafka.producer.topicPrefix", "sv-uploads-")
	cfg.Set("kafka.producer.partitioner", partitioner)
	fwd, err := NewKafkaForwarder(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer fwd.Close()

	var mu sync.Mutex
	var wg sync.WaitGroup
	counts := map[int32]int{}
	for i := 0; i < messages; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			partition, _, err := fwd.Produce(context.Background(), "topic", []byte("event"))
			if err != nil {
				t.Errorf("produce failed: %v", err)
				return
			}
			mu.Lock()
			counts[partition]++
			mu.Unlock()
		}()
	}
	wg.Wait()
	return counts
}

func TestProducerSpreadsUnkeyedEventsWithHashPartitioner(t *testing.T) {
	counts := producedPartitions(t, "hash", 50)
	if len(counts) < 2 {
		t.Fatalf("expected events on several partitions, got %v", counts)
	}
}

func TestProducerKeepsConcurrentEventsTogetherWithStickyPartitioner(t *testing.T) {
	counts := producedPartitions(t, "sticky", 50)
	if len(counts) != 1 {
		t.Fatalf("expected every event on one partition, got %v", counts)
	}
}
