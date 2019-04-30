package mocks

import (
	"context"
	"sync"

	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
)

type MockSender struct {
	totalSent           int
	totalCalls          int
	failureIndexesOrder [][]int64
	firstCallEvents     []*pb.Event
	eventsIdsFreqs      map[string]int
	mutex               *sync.Mutex
}

func NewMockSender() *MockSender {
	return &MockSender{
		failureIndexesOrder: [][]int64{},
		eventsIdsFreqs:      map[string]int{},
		mutex:               &sync.Mutex{},
	}
}

func (ms *MockSender) SendEvents(
	ctx context.Context,
	events []*pb.Event,
) []int64 {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	defer func() { ms.totalCalls++ }()
	ms.totalSent += len(events)
	if ms.totalCalls == 0 {
		ms.firstCallEvents = events
	}
	for _, e := range events {
		ms.eventsIdsFreqs[e.Id]++
	}
	if len(ms.failureIndexesOrder) > ms.totalCalls {
		ret := ms.failureIndexesOrder[ms.totalCalls]
		ms.totalSent -= len(ret)
		return ret
	}
	return []int64{}
}

func (ms *MockSender) SendEvent(
	ctx context.Context,
	event *pb.Event,
) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.totalCalls++
	ms.totalSent++
	return nil
}

func (ms *MockSender) SetFailureIndexesOrder(failureIndexesOrder [][]int64) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.failureIndexesOrder = failureIndexesOrder
}

func (ms MockSender) GetTotalSent() int {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	return ms.totalSent
}

func (ms MockSender) GetTotalCalls() int {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	return ms.totalCalls
}

func (ms MockSender) GetFirstCallEvents() []*pb.Event {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	return ms.firstCallEvents
}

func (ms MockSender) GetEventsIdsFreqs() map[string]int {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	return ms.eventsIdsFreqs
}
