// eventsgateway
// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2019 Top Free Games <backend@tfgco.com>

package client

import (
	"context"

	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
)

type GRPCClient interface {
	send(context.Context, *pb.Event) error
	GracefulStop() error
}
