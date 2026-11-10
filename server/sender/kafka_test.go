//go:build unit

package sender

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/v4/server/logger"
	"github.com/topfreegames/eventsgateway/v4/server/metrics"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
)

type fakeForwarder struct {
	mu       sync.Mutex
	fail     func(topic string, message []byte) bool
	produced [][]byte
}

func (f *fakeForwarder) Produce(_ context.Context, topic string, message []byte) (int32, int64, error) {
	if f.fail != nil && f.fail(topic, message) {
		return 0, 0, errors.New("produce failed")
	}
	f.mu.Lock()
	f.produced = append(f.produced, message)
	f.mu.Unlock()
	return 0, 0, nil
}

var metricsOnce sync.Once

func testConfig() *viper.Viper {
	cfg := viper.New()
	cfg.Set("kafka.producer.maxMessageBytes", 1000000)
	cfg.Set("kafka.producer.topicPrefix", "sv-uploads-")
	cfg.Set("prometheus.enabled", "false")
	metricsOnce.Do(func() { metrics.StartServer(cfg) })
	return cfg
}

func testEvents(n int) []*pb.Event {
	events := make([]*pb.Event, n)
	for i := range events {
		events[i] = &pb.Event{
			Id:        "id",
			Name:      "name",
			Topic:     "topic",
			Props:     map[string]string{"k": "v"},
			Timestamp: time.Now().UnixMilli(),
		}
	}
	return events
}

func TestSendEventsReportsEveryFailedIndex(t *testing.T) {
	cfg := testConfig()
	fwd := &fakeForwarder{fail: func(string, []byte) bool { return true }}
	s := NewKafkaSender(fwd, &logger.NullLogger{}, cfg)
	for iter := 0; iter < 300; iter++ {
		events := testEvents(50)
		got := s.SendEvents(context.Background(), events)
		if len(got) != len(events) {
			t.Fatalf("iteration %d: expected %d failure indexes, got %d", iter, len(events), len(got))
		}
		sorted := append([]int64(nil), got...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
		for i, idx := range sorted {
			if idx != int64(i) {
				t.Fatalf("iteration %d: expected index %d, got %d", iter, i, idx)
			}
		}
	}
}

func TestSendEventsReportsOnlyFailedIndexes(t *testing.T) {
	cfg := testConfig()
	events := testEvents(10)
	events[3].Name = "bad"
	events[7].Name = "bad"
	fwd := &fakeForwarder{fail: func(_ string, m []byte) bool { return containsName(m, "bad") }}
	s := NewKafkaSender(fwd, &logger.NullLogger{}, cfg)
	got := s.SendEvents(context.Background(), events)
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	if len(got) != 2 || got[0] != 3 || got[1] != 7 {
		t.Fatalf("expected failure indexes [3 7], got %v", got)
	}
	if len(fwd.produced) != 8 {
		t.Fatalf("expected 8 produced events, got %d", len(fwd.produced))
	}
}

func TestSendEventsWithInvalidEventReportsItsIndex(t *testing.T) {
	cfg := testConfig()
	events := testEvents(3)
	events[1].Id = ""
	fwd := &fakeForwarder{}
	s := NewKafkaSender(fwd, &logger.NullLogger{}, cfg)
	got := s.SendEvents(context.Background(), events)
	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("expected failure indexes [1], got %v", got)
	}
}

func TestSendEventsEmptyBatch(t *testing.T) {
	cfg := testConfig()
	s := NewKafkaSender(&fakeForwarder{}, &logger.NullLogger{}, cfg)
	if got := s.SendEvents(context.Background(), nil); len(got) != 0 {
		t.Fatalf("expected no failure indexes, got %v", got)
	}
}

func containsName(message []byte, name string) bool {
	for i := 0; i+len(name) <= len(message); i++ {
		if string(message[i:i+len(name)]) == name {
			return true
		}
	}
	return false
}
