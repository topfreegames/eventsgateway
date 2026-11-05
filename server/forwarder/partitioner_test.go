//go:build unit

package forwarder

import (
	"sync"
	"testing"

	"github.com/IBM/sarama"
)

func message(size int) *sarama.ProducerMessage {
	return &sarama.ProducerMessage{Topic: "t", Value: sarama.ByteEncoder(make([]byte, size))}
}

func TestStickyPartitionerKeepsPartitionUntilByteLimit(t *testing.T) {
	p := NewStickyPartitionerConstructor(1000)("t")
	first, err := p.Partition(message(100), 12)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 9; i++ {
		got, _ := p.Partition(message(100), 12)
		if got != first {
			t.Fatalf("message %d: expected partition %d, got %d", i+2, first, got)
		}
	}
	next, _ := p.Partition(message(100), 12)
	if next == first {
		t.Fatalf("expected a different partition after %d bytes, stayed on %d", 1000, first)
	}
}

func TestStickyPartitionerResetsWhenPartitionCountChanges(t *testing.T) {
	p := NewStickyPartitionerConstructor(1 << 30)("t")
	for i := 0; i < 50; i++ {
		got, _ := p.Partition(message(10), 12)
		if got < 0 || got >= 12 {
			t.Fatalf("partition out of range: %d", got)
		}
	}
	for i := 0; i < 50; i++ {
		got, _ := p.Partition(message(10), 3)
		if got < 0 || got >= 3 {
			t.Fatalf("partition out of range after shrink: %d", got)
		}
	}
}

func TestStickyPartitionerSinglePartition(t *testing.T) {
	p := NewStickyPartitionerConstructor(10)("t")
	for i := 0; i < 20; i++ {
		got, err := p.Partition(message(100), 1)
		if err != nil || got != 0 {
			t.Fatalf("expected partition 0, got %d err %v", got, err)
		}
	}
}

func TestStickyPartitionerRejectsZeroPartitions(t *testing.T) {
	p := NewStickyPartitionerConstructor(10)("t")
	if _, err := p.Partition(message(1), 0); err == nil {
		t.Fatal("expected an error for zero partitions")
	}
}

func TestStickyPartitionerSpreadsLoadOverTime(t *testing.T) {
	const partitions = 12
	p := NewStickyPartitionerConstructor(1000)("t")
	counts := make([]int, partitions)
	for i := 0; i < 120000; i++ {
		got, _ := p.Partition(message(100), partitions)
		counts[got]++
	}
	expected := 120000 / partitions
	for i, c := range counts {
		if c < expected/2 || c > expected*3/2 {
			t.Fatalf("partition %d got %d messages, expected about %d: %v", i, c, expected, counts)
		}
	}
}

func TestStickyPartitionerKeyedMessagesMatchHashPartitioner(t *testing.T) {
	p := NewStickyPartitionerConstructor(10)("t")
	h := sarama.NewHashPartitioner("t")
	for _, key := range []string{"a", "b", "player-1", "player-2"} {
		msg := &sarama.ProducerMessage{Topic: "t", Key: sarama.StringEncoder(key), Value: sarama.ByteEncoder("v")}
		got, _ := p.Partition(msg, 12)
		want, _ := h.Partition(msg, 12)
		if got != want {
			t.Fatalf("key %q: expected %d, got %d", key, want, got)
		}
	}
}

func TestStickyPartitionerIsSafeForConcurrentUse(t *testing.T) {
	p := NewStickyPartitionerConstructor(500)("t")
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				got, _ := p.Partition(message(50), 6)
				if got < 0 || got >= 6 {
					t.Errorf("partition out of range: %d", got)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestStickyPartitionerDoesNotRequireConsistency(t *testing.T) {
	if NewStickyPartitionerConstructor(10)("t").RequiresConsistency() {
		t.Fatal("expected RequiresConsistency to be false")
	}
}
