package sender

import (
	"context"

	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
)

type Sender interface {
	SendEvents(context.Context, []*pb.Event) []int64
	SendEvent(context.Context, *pb.Event) error
}
