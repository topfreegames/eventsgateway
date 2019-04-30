package client

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/metrics"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
	"google.golang.org/grpc"
)

type gRPCClientSync struct {
	client pb.GRPCForwarderClient
	config *viper.Viper
	conn   *grpc.ClientConn
	logger logrus.FieldLogger
	wg     sync.WaitGroup
}

func newGRPCClientSync(
	configPrefix string,
	config *viper.Viper,
	logger logrus.FieldLogger,
	serverAddress string,
	client pb.GRPCForwarderClient,
	opts ...grpc.DialOption,
) (*gRPCClientSync, error) {
	g := &gRPCClientSync{
		config: config,
		logger: logger,
	}
	if err := g.configureGRPCForwarderClient(
		serverAddress,
		client,
		opts...,
	); err != nil {
		return nil, err
	}
	return g, nil
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
	s.logger.WithFields(logrus.Fields{
		"operation": "configureGRPCForwarderClient",
	}).Info("connecting to grpc server")
	dialOpts := append(
		[]grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithUnaryInterceptor(s.metricsReporterInterceptor),
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
	l := s.logger.WithFields(logrus.Fields{
		"method": method,
	})

	defer func(startTime time.Time) {
		timeUsed := float64(time.Since(startTime).Nanoseconds() / 1000000)
		metrics.ClientRequestsResponseTime.WithLabelValues(hostname, method).Observe(timeUsed)
		l.WithFields(logrus.Fields{
			"timeUsed": timeUsed,
			"reply":    reply.(*pb.SendEventResponse),
		}).Debug("request processed")
	}(time.Now())

	err := invoker(ctx, method, req, reply, cc, opts...)
	if err != nil {
		l.WithError(err).Error("error processing request")
		metrics.ClientRequestsFailureCounter.WithLabelValues(hostname, method, err.Error()).Inc()
	} else {
		metrics.ClientRequestsSuccessCounter.WithLabelValues(hostname, method).Inc()
	}
	return err
}

func (s *gRPCClientSync) send(ctx context.Context, event *pb.Event) error {
	_, err := s.client.SendEvent(ctx, event)
	return err
}

// Wait on pending async send of events
func (s *gRPCClientSync) Wait() {
	s.wg.Wait()
	if err := s.conn.Close(); err != nil {
		panic(err.Error())
	}
}
