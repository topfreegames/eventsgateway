// eventsgateway
// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2019 Top Free Games <backend@tfgco.com>

package mocks

import (
	"context"
	"sync"

	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
)

type MockSender struct {
	mu                  *sync.Mutex
	totalSent           int
	totalCalls          int
	failureIndexesOrder [][]int64
	firstCallEvents     []*pb.Event
	eventsIdsFreqs      map[string]int
}

func NewMockSender() *MockSender {
	return &MockSender{
		failureIndexesOrder: [][]int64{},
		eventsIdsFreqs:      map[string]int{},
		mu:                  &sync.Mutex{},
	}
}

func (ms *MockSender) SendEvents(
	ctx context.Context,
	events []*pb.Event,
) []int64 {
	ms.mu.Lock()
	defer func() {
		ms.totalCalls++
		ms.mu.Unlock()
	}()
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
	return nil
}

func (ms *MockSender) SendEvent(
	ctx context.Context,
	event *pb.Event,
) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.totalCalls++
	ms.totalSent++
	return nil
}

func (ms *MockSender) SetFailureIndexesOrder(failureIndexesOrder [][]int64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.failureIndexesOrder = failureIndexesOrder
}

func (ms MockSender) GetTotalSent() int {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.totalSent
}

func (ms MockSender) GetTotalCalls() int {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.totalCalls
}

func (ms MockSender) GetFirstCallEvents() []*pb.Event {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.firstCallEvents
}

func (ms MockSender) GetEventsIdsFreqs() map[string]int {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.eventsIdsFreqs
}
