package forwarder

import (
	"context"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/opentracing/opentracing-go"
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
	"time"
)

type KafkaForwarder struct {
	producer    sarama.SyncProducer
	topicPrefix string
}

func NewKafkaForwarder(config *viper.Viper) (*KafkaForwarder, error) {
	if config.GetBool("kafka.logger.enabled") {
		sarama.Logger = log.New(os.Stdout, "sarama", log.Llongfile)
	}

	kafkaConf := sarama.NewConfig()
	kafkaConf.Net.MaxOpenRequests = config.GetInt("kafka.producer.net.maxOpenRequests")
	kafkaConf.Net.DialTimeout = config.GetDuration("kafka.producer.net.dialTimeout")
	kafkaConf.Net.ReadTimeout = config.GetDuration("kafka.producer.net.readTimeout")
	kafkaConf.Net.WriteTimeout = config.GetDuration("kafka.producer.net.writeTimeout")
	kafkaConf.Net.KeepAlive = config.GetDuration("kafka.producer.net.keepAlive")
	kafkaConf.Producer.Return.Errors = true
	kafkaConf.Producer.Return.Successes = true
	kafkaConf.Producer.Idempotent = config.GetBool("kafka.producer.idempotent")
	kafkaConf.Producer.MaxMessageBytes = config.GetInt("kafka.producer.maxMessageBytes")
	kafkaConf.Producer.Timeout = config.GetDuration("kafka.producer.timeout")
	kafkaConf.Producer.Flush.Bytes = config.GetInt("kafka.producer.batch.size")
	kafkaConf.Producer.Flush.Frequency = time.Duration(config.GetInt("kafka.producer.linger.ms")) * time.Millisecond
	kafkaConf.Producer.Retry.Max = config.GetInt("kafka.producer.retry.max")
	kafkaConf.Producer.RequiredAcks = sarama.WaitForLocal
	kafkaConf.Producer.Compression = sarama.CompressionSnappy
	kafkaConf.ClientID = config.GetString("kafka.producer.clientId")
	kafkaConf.Version = sarama.V2_2_0_0

	brokers := strings.Split(config.GetString("kafka.producer.brokers"), ",")
	producer, err := sarama.NewSyncProducer(brokers, kafkaConf)
	if err != nil {
		return nil, err
	}

	topicPrefix := config.GetString("kafka.producer.topicPrefix")

	return &KafkaForwarder{
		producer:    producer,
		topicPrefix: topicPrefix,
	}, nil
}

func (k KafkaForwarder) Produce(ctx context.Context, topic string, message []byte) (int32, int64, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "forwarder.kafka.Produce")
	span.SetTag("kafkaTopic", topic)
	defer span.Finish()

	prefixedTopic := fmt.Sprintf("%s%s", k.topicPrefix, topic)
	kafkaMsg := &sarama.ProducerMessage{
		Topic: prefixedTopic,
		Value: sarama.ByteEncoder(message),
	}

	partition, offset, err := k.producer.SendMessage(kafkaMsg)
	return partition, offset, err
}
