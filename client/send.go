// eventsgateway
// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2018 Top Free Games <backend@tfgco.com>

package client

import (
	"math"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
	context "golang.org/x/net/context"
)

// Send sends an event to another server via grpc using the client's configured topic
func (g *GRPCClient) Send(name string, props map[string]string) error {
	l := g.logger.WithFields(log.Fields{
		"operation": "send",
		"event":     name,
	})
	l.Debug("sending event")
	if err := g.sendEvent(name, g.topic, props); err != nil {
		l.WithError(err).Error("send event failed")
		return err
	}
	return nil
}

// SendToTopic sends an event to another server via grpc using an explicit topic
func (g *GRPCClient) SendToTopic(name, topic string, props map[string]string) error {
	l := g.logger.WithFields(log.Fields{
		"operation": "sendToTopic",
		"event":     name,
		"topic":     topic,
	})
	l.Debug("sending event")
	if err := g.sendEvent(name, topic, props); err != nil {
		l.WithError(err).Error("send event failed")
		return err
	}
	return nil
}

func (g *GRPCClient) sendEvent(name, topic string, props map[string]string) error {
	g.logger.WithFields(log.Fields{
		"operation": "eventRequest",
		"name":      name,
	}).Debug("getting event request")

	event := &pb.Event{
		Id:        uuid.NewV4().String(),
		Name:      name,
		Topic:     topic,
		Props:     props,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	}

	if g.async {
		g.eventsChannel <- event
		return nil
	} else {
		ctx, _ := context.WithTimeout(context.Background(), g.timeout)
		_, err := g.client.SendEvent(ctx, event)
		return err
	}
}

func (g *GRPCClient) sendEventsRoutine() {
	ticker := time.NewTicker(g.flushInterval)
	defer ticker.Stop()

	msg := &pb.Events{}
	msg.Events = make([]*pb.Event, 0, g.flushSize)

	send := func() {
		cpy := msg
		cpy.Id = uuid.NewV4().String()
		msg = &pb.Events{}
		msg.Events = make([]*pb.Event, 0, g.flushSize)
		go g.sendEvents(cpy, 0)
	}

	for {
		select {
		case e := <-g.eventsChannel:
			if len(msg.Events) == 0 {
				g.wg.Add(1)
			}
			msg.Events = append(msg.Events, e)
			if len(msg.Events) == g.flushSize {
				send()
			}
		case <-ticker.C:
			if len(msg.Events) > 0 {
				send()
			}
		}
	}
}

func (g *GRPCClient) sendEvents(msg *pb.Events, retryCount int) {
	if retryCount == g.asyncRetries {
		g.logger.WithFields(logrus.Fields{
			"id":           msg.Id,
			"asyncRetries": g.asyncRetries,
			"size":         len(msg.Events),
		}).Infof("dropped events due to max retries")
		g.wg.Done()
		return
	}
	ctx, _ := context.WithTimeout(context.Background(), g.timeout)
	if res, err := g.client.SendEvents(ctx, msg); err != nil {
		g.logger.WithFields(logrus.Fields{
			"id":           msg.Id,
			"size":         len(msg.Events),
			"failureIndex": res.FailureIndex,
			"retryCount":   retryCount,
		}).WithError(err).Error("failed to send events")
		time.Sleep(time.Duration(math.Pow(2, float64(retryCount))) * g.asyncRetryInterval)
		msg.Events = msg.Events[res.FailureIndex-1:]
		g.sendEvents(msg, retryCount+1)
	} else {
		g.wg.Done()
	}
}
