// eventsgateway
// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2019 Top Free Games <backend@tfgco.com>

package sender

import (
	"context"

	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
)

type Sender interface {
	SendEvents(context.Context, []*pb.Event) []int64
	SendEvent(context.Context, *pb.Event) error
}
