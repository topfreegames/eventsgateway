package forwarder

import (
	"math/rand"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

type stickyPartitioner struct {
	mu            sync.Mutex
	maxBytes      int
	current       int32
	accumulated   int
	numPartitions int32
	rand          *rand.Rand
	keyed         sarama.Partitioner
}

func NewStickyPartitionerConstructor(maxBytes int) sarama.PartitionerConstructor {
	return func(topic string) sarama.Partitioner {
		return &stickyPartitioner{
			maxBytes: maxBytes,
			current:  -1,
			rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
			keyed:    sarama.NewHashPartitioner(topic),
		}
	}
}

func (p *stickyPartitioner) Partition(message *sarama.ProducerMessage, numPartitions int32) (int32, error) {
	if message.Key != nil {
		return p.keyed.Partition(message, numPartitions)
	}
	if numPartitions <= 0 {
		return -1, sarama.ErrInvalidPartition
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.current < 0 || numPartitions != p.numPartitions || p.accumulated >= p.maxBytes {
		p.current = p.nextPartition(numPartitions)
		p.numPartitions = numPartitions
		p.accumulated = 0
	}
	if message.Value != nil {
		p.accumulated += message.Value.Length()
	}
	return p.current, nil
}

func (p *stickyPartitioner) nextPartition(numPartitions int32) int32 {
	if numPartitions == 1 {
		return 0
	}
	candidate := p.rand.Int31n(numPartitions)
	if candidate == p.current {
		candidate = (candidate + 1) % numPartitions
	}
	return candidate
}

func (p *stickyPartitioner) RequiresConsistency() bool {
	return false
}
