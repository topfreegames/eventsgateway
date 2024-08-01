// eventsgateway
// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2019 Top Free Games <backend@tfgco.com>

package client

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"reflect"
	"time"
	"unsafe"

	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/v4/logger"
	"github.com/topfreegames/eventsgateway/v4/metrics"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
	"google.golang.org/grpc"
)

type gRPCClientSync struct {
	client  pb.GRPCForwarderClient
	config  *viper.Viper
	conn    *grpc.ClientConn
	logger  logger.Logger
	timeout time.Duration
}

func newGRPCClientSync(
	configPrefix string,
	config *viper.Viper,
	logger logger.Logger,
	serverAddress string,
	client pb.GRPCForwarderClient,
	opts ...grpc.DialOption,
) (*gRPCClientSync, error) {
	s := &gRPCClientSync{
		config: config,
		logger: logger,
	}
	timeoutConf := fmt.Sprintf("%sclient.grpc.timeout", configPrefix)
	s.config.SetDefault(timeoutConf, 500*time.Millisecond)
	s.timeout = s.config.GetDuration(timeoutConf)
	s.logger = logger.WithFields(map[string]interface{}{
		"timeout": s.timeout,
	})
	if err := s.configureGRPCForwarderClient(
		serverAddress,
		client,
		opts...,
	); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *gRPCClientSync) configureGRPCForwarderClient(
	serverAddress string,
	client pb.GRPCForwarderClient,
	opts ...grpc.DialOption,
) error {
	if client != nil {
		s.client = client
		return nil
	}
	s.logger.WithFields(map[string]interface{}{
		"operation": "configureGRPCForwarderClient",
	}).Info("connecting to grpc server")

	dialOpts := append(
		[]grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithChainUnaryInterceptor(
				s.metricsReporterInterceptor,
			),
		},
		opts...,
	)

	conn, err := grpc.Dial(serverAddress, dialOpts...)

	if err != nil {
		return err
	}
	s.conn = conn
	s.client = pb.NewGRPCForwarderClient(conn)
	return nil
}

// metricsReporterInterceptor will report metrics from client
func (s *gRPCClientSync) metricsReporterInterceptor(
	ctx context.Context,
	method string,
	req interface{},
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	l := s.logger.WithFields(map[string]interface{}{
		"method": method,
	})

	event := req.(*pb.Event)

	defer func(startTime time.Time) {
		elapsedTime := float64(time.Since(startTime).Nanoseconds() / 1000000)
		metrics.ClientRequestsResponseTime.WithLabelValues(
			method,
			event.Topic,
			"0",
		).Observe(elapsedTime)
		l.WithFields(map[string]interface{}{
			"elapsedTime": elapsedTime,
			"reply":       reply.(*pb.SendEventResponse),
		}).Debug("request processed")
	}(time.Now())

	if err := invoker(ctx, method, req, reply, cc, opts...); err != nil {
		l.WithError(err).Error("error processing request")
		metrics.ClientRequestsFailureCounter.WithLabelValues(
			method,
			event.Topic,
			"0",
			err.Error(),
		).Inc()
		return err
	}
	metrics.ClientRequestsSuccessCounter.WithLabelValues(
		method,
		event.Topic,
		"0",
	).Inc()
	return nil
}

func printContextInternals(ctx interface{}, inner bool) {
	contextValues := reflect.ValueOf(ctx).Elem()
	contextKeys := reflect.TypeOf(ctx).Elem()

	if !inner {
		fmt.Printf("\nFields for %s.%s\n", contextKeys.PkgPath(), contextKeys.Name())
	}

	if contextKeys.Kind() == reflect.Struct {
		for i := 0; i < contextValues.NumField(); i++ {
			reflectValue := contextValues.Field(i)
			reflectValue = reflect.NewAt(reflectValue.Type(), unsafe.Pointer(reflectValue.UnsafeAddr())).Elem()

			reflectField := contextKeys.Field(i)

			if reflectField.Name == "Context" {
				printContextInternals(reflectValue.Interface(), true)
			} else {
				fmt.Printf("field name: %+v\n", reflectField.Name)
				fmt.Printf("value: %+v\n", reflectValue.Interface())
			}
		}
	} else {
		fmt.Printf("context is empty (int)\n")
	}
}

func (s *gRPCClientSync) send(ctx context.Context, event *pb.Event) error {
	childCtx, span := otel.Tracer("client.sync").Start(ctx, "client.sync.send")
	defer span.End()
	//printContextInternals(ctx, true)
	ctxWithTimeout, cancel := context.WithTimeout(childCtx, s.timeout)
	defer cancel()
	_, err := s.client.SendEvent(ctxWithTimeout, event)
	return err
}

// GracefulStop closes client connection
func (s *gRPCClientSync) GracefulStop() error {
	return s.conn.Close()
}
