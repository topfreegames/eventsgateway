// MIT License
//
// Copyright (c) 2018 Top Free Games
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package app

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"github.com/topfreegames/eventsgateway/v4/server/logger"
	"github.com/topfreegames/eventsgateway/v4/server/sender"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
)

// Server struct
type Server struct {
	logger logger.Logger
	sender sender.Sender
}

// NewServer returns a new grpc server
func NewServer(sender sender.Sender, logger logger.Logger) *Server {
	s := &Server{
		logger: logger,
		sender: sender,
	}
	return s
}

func (s *Server) SendEvent(
	ctx context.Context,
	req *pb.Event,
) (*pb.SendEventResponse, error) {
	span, childCtx := opentracing.StartSpanFromContext(ctx, "app.server.SendEvent")
	defer span.Finish()
	if err := s.sender.SendEvent(childCtx, req); err != nil {
		return nil, err
	}
	return &pb.SendEventResponse{}, nil
}

// SendEvents response might include FailureIndexes in case producer fails
// to send all events
func (s *Server) SendEvents(
	ctx context.Context,
	req *pb.SendEventsRequest,
) (*pb.SendEventsResponse, error) {
	span, childCtx := opentracing.StartSpanFromContext(ctx, "app.server.SendEvents")
	span.SetTag("eventCount", len(req.Events))
	defer span.Finish()

	failureIndexes := s.sender.SendEvents(childCtx, req.Events)
	return &pb.SendEventsResponse{FailureIndexes: failureIndexes}, nil
}
