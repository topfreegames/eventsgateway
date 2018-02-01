package app

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/Shopify/sarama"
	"github.com/spf13/viper"
	"github.com/topfreegames/eventsgateway/forwarder"
	extensions "github.com/topfreegames/extensions/kafka"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"

	"github.com/sirupsen/logrus"
)

// App is the app structure
type App struct {
	host            string
	port            int
	server          *Server
	config          *viper.Viper
	log             logrus.FieldLogger
	eventsForwarder forwarder.Forwarder
}

// NewApp creates a new App object
func NewApp(host string, port int, log logrus.FieldLogger, config *viper.Viper) (*App, error) {
	a := &App{
		host:   host,
		port:   port,
		log:    log,
		config: config,
	}
	err := a.configure()
	return a, err
}

func (a *App) loadConfigurationDefaults() {
	a.config.SetDefault("extensions.kafkaproducer.brokers", "localhost:9192")
	a.config.SetDefault("extensions.kafkaproducer.maxMessageBytes", 3000000)
}

func (a *App) configure() error {
	a.loadConfigurationDefaults()
	err := a.configureEventsForwarder()
	if err != nil {
		return err
	}
	return nil
}

func (a *App) configureEventsForwarder() error {
	kafkaConf := sarama.NewConfig()
	kafkaConf.Producer.Return.Errors = true
	kafkaConf.Producer.Return.Successes = true
	kafkaConf.Producer.MaxMessageBytes = a.config.GetInt("extensions.kafkaproducer.maxMessageBytes")
	kafkaConf.Producer.RequiredAcks = sarama.WaitForLocal
	kafkaConf.Producer.Compression = sarama.CompressionSnappy
	k, err := extensions.NewSyncProducer(a.config, a.log.(*logrus.Logger), kafkaConf)
	if err != nil {
		return err
	}
	a.server = NewServer(k, a.log)
	return nil
}

// Run runs the app
func (a *App) Run() {
	log := a.log
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", a.host, a.port))
	if err != nil {
		log.Panic(err.Error())
	}
	log.Infof("events gateway listening on %s:%d", a.host, a.port)
	grpcServer := grpc.NewServer()
	pb.RegisterGRPCForwarderServer(grpcServer, a.server)

	grpcServer.Serve(listener)
}
