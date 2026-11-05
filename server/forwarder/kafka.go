package forwarder

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type KafkaForwarder struct {
	producer    sarama.SyncProducer
	topicPrefix string
}

func NewKafkaForwarder(config *viper.Viper) (*KafkaForwarder, error) {
	if config.GetBool("kafka.logger.enabled") {
		sarama.Logger = log.New(os.Stdout, "sarama", log.Llongfile)
	}

	kafkaConf, err := newSaramaConfig(config)
	if err != nil {
		return nil, err
	}

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

func setProducerDefaults(config *viper.Viper) {
	config.SetDefault("kafka.producer.compression", "snappy")
	config.SetDefault("kafka.producer.compressionLevel", sarama.CompressionLevelDefault)
	config.SetDefault("kafka.producer.partitioner", "hash")
	config.SetDefault("kafka.producer.stickyPartitionerBytes", 65536)
}

func newSaramaConfig(config *viper.Viper) (*sarama.Config, error) {
	setProducerDefaults(config)

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
	kafkaConf.ClientID = config.GetString("kafka.producer.clientId")
	kafkaConf.Version = sarama.V3_7_1_0

	codec, err := parseCompression(config.GetString("kafka.producer.compression"))
	if err != nil {
		return nil, err
	}
	kafkaConf.Producer.Compression = codec
	kafkaConf.Producer.CompressionLevel = config.GetInt("kafka.producer.compressionLevel")

	switch partitioner := strings.ToLower(config.GetString("kafka.producer.partitioner")); partitioner {
	case "hash":
		kafkaConf.Producer.Partitioner = sarama.NewHashPartitioner
	case "sticky":
		kafkaConf.Producer.Partitioner = NewStickyPartitionerConstructor(config.GetInt("kafka.producer.stickyPartitionerBytes"))
	default:
		return nil, fmt.Errorf("unknown kafka.producer.partitioner %q", partitioner)
	}

	if err := kafkaConf.Validate(); err != nil {
		return nil, err
	}
	return kafkaConf, nil
}

func parseCompression(name string) (sarama.CompressionCodec, error) {
	switch strings.ToLower(name) {
	case "none":
		return sarama.CompressionNone, nil
	case "gzip":
		return sarama.CompressionGZIP, nil
	case "snappy":
		return sarama.CompressionSnappy, nil
	case "lz4":
		return sarama.CompressionLZ4, nil
	case "zstd":
		return sarama.CompressionZSTD, nil
	default:
		return sarama.CompressionNone, fmt.Errorf("unknown kafka.producer.compression %q", name)
	}
}

func (k KafkaForwarder) Produce(ctx context.Context, topic string, message []byte) (int32, int64, error) {
	_, span := otel.Tracer("forwarder.kafka").Start(ctx, "forwarder.kafka.Produce")
	span.SetAttributes(attribute.Key("kafkaTopic").String(topic))
	defer span.End()

	prefixedTopic := fmt.Sprintf("%s%s", k.topicPrefix, topic)
	kafkaMsg := &sarama.ProducerMessage{
		Topic: prefixedTopic,
		Value: sarama.ByteEncoder(message),
	}

	partition, offset, err := k.producer.SendMessage(kafkaMsg)
	return partition, offset, err
}

func (k KafkaForwarder) Close() error {
	return k.producer.Close()
}
