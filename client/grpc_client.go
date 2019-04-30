package client

import (
	"context"

	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
)

type GRPCClient interface {
	send(context.Context, *pb.Event) error
	Wait()
}
