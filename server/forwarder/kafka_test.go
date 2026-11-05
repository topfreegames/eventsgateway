//go:build unit

package forwarder

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/spf13/viper"
)

func appDefaults() *viper.Viper {
	cfg := viper.New()
	cfg.SetDefault("kafka.producer.net.maxOpenRequests", 10)
	cfg.SetDefault("kafka.producer.net.dialTimeout", "500ms")
	cfg.SetDefault("kafka.producer.net.readTimeout", "250ms")
	cfg.SetDefault("kafka.producer.net.writeTimeout", "250ms")
	cfg.SetDefault("kafka.producer.idempotent", false)
	cfg.SetDefault("kafka.producer.net.keepAlive", "60s")
	cfg.SetDefault("kafka.producer.maxMessageBytes", 1000000)
	cfg.SetDefault("kafka.producer.timeout", "250ms")
	cfg.SetDefault("kafka.producer.batch.size", 1000000)
	cfg.SetDefault("kafka.producer.linger.ms", 1)
	cfg.SetDefault("kafka.producer.retry.max", 0)
	cfg.SetDefault("kafka.producer.clientId", "eventsgateway")
	return cfg
}

func partitionerType(c sarama.PartitionerConstructor) string {
	return fmt.Sprintf("%T", c("t"))
}

func TestSaramaConfigDefaultsMatchPreviousBehavior(t *testing.T) {
	conf, err := newSaramaConfig(appDefaults())
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name      string
		got, want interface{}
	}{
		{"MaxOpenRequests", conf.Net.MaxOpenRequests, 10},
		{"DialTimeout", conf.Net.DialTimeout, 500 * time.Millisecond},
		{"ReadTimeout", conf.Net.ReadTimeout, 250 * time.Millisecond},
		{"WriteTimeout", conf.Net.WriteTimeout, 250 * time.Millisecond},
		{"KeepAlive", conf.Net.KeepAlive, 60 * time.Second},
		{"Idempotent", conf.Producer.Idempotent, false},
		{"MaxMessageBytes", conf.Producer.MaxMessageBytes, 1000000},
		{"Timeout", conf.Producer.Timeout, 250 * time.Millisecond},
		{"FlushBytes", conf.Producer.Flush.Bytes, 1000000},
		{"FlushFrequency", conf.Producer.Flush.Frequency, time.Millisecond},
		{"RetryMax", conf.Producer.Retry.Max, 0},
		{"RetryBackoff", conf.Producer.Retry.Backoff, sarama.NewConfig().Producer.Retry.Backoff},
		{"RequiredAcks", conf.Producer.RequiredAcks, sarama.WaitForLocal},
		{"Compression", conf.Producer.Compression, sarama.CompressionSnappy},
		{"CompressionLevel", conf.Producer.CompressionLevel, sarama.CompressionLevelDefault},
		{"ReturnSuccesses", conf.Producer.Return.Successes, true},
		{"ReturnErrors", conf.Producer.Return.Errors, true},
		{"ClientID", conf.ClientID, "eventsgateway"},
		{"Version", conf.Version, sarama.V3_7_1_0},
		{"Partitioner", partitionerType(conf.Producer.Partitioner), partitionerType(sarama.NewHashPartitioner)},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestSaramaConfigAppliesTunedValues(t *testing.T) {
	cfg := appDefaults()
	cfg.Set("kafka.producer.linger.ms", 50)
	cfg.Set("kafka.producer.compression", "zstd")
	cfg.Set("kafka.producer.compressionLevel", 1)
	cfg.Set("kafka.producer.partitioner", "sticky")
	cfg.Set("kafka.producer.stickyPartitionerBytes", 4096)
	conf, err := newSaramaConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if conf.Producer.Flush.Frequency != 50*time.Millisecond {
		t.Errorf("FlushFrequency: got %v", conf.Producer.Flush.Frequency)
	}
	if conf.Producer.Compression != sarama.CompressionZSTD || conf.Producer.CompressionLevel != 1 {
		t.Errorf("Compression: got %v level %d", conf.Producer.Compression, conf.Producer.CompressionLevel)
	}
	p := conf.Producer.Partitioner("t")
	sticky, ok := p.(*stickyPartitioner)
	if !ok {
		t.Fatalf("expected sticky partitioner, got %T", p)
	}
	if sticky.maxBytes != 4096 {
		t.Errorf("stickyPartitionerBytes: got %d", sticky.maxBytes)
	}
}

func TestSaramaConfigAcceptsEveryCompression(t *testing.T) {
	for _, name := range []string{"none", "gzip", "snappy", "lz4", "zstd", "ZSTD"} {
		cfg := appDefaults()
		cfg.Set("kafka.producer.compression", name)
		if _, err := newSaramaConfig(cfg); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestSaramaConfigRejectsUnknownValues(t *testing.T) {
	cfg := appDefaults()
	cfg.Set("kafka.producer.compression", "brotli")
	if _, err := newSaramaConfig(cfg); err == nil {
		t.Error("expected an error for unknown compression")
	}
	cfg = appDefaults()
	cfg.Set("kafka.producer.partitioner", "roundrobin")
	if _, err := newSaramaConfig(cfg); err == nil {
		t.Error("expected an error for unknown partitioner")
	}
}

func TestSaramaConfigRejectsInvalidCombination(t *testing.T) {
	cfg := appDefaults()
	cfg.Set("kafka.producer.idempotent", true)
	if _, err := newSaramaConfig(cfg); err == nil {
		t.Error("expected idempotent producer without retries and acks=all to be rejected")
	}
}

func TestSaramaConfigReadsEnvOverrides(t *testing.T) {
	t.Setenv("EVENTSGATEWAY_KAFKA_PRODUCER_COMPRESSION", "lz4")
	t.Setenv("EVENTSGATEWAY_KAFKA_PRODUCER_PARTITIONER", "sticky")
	cfg := appDefaults()
	cfg.SetEnvPrefix("eventsgateway")
	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	cfg.AutomaticEnv()
	conf, err := newSaramaConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if conf.Producer.Compression != sarama.CompressionLZ4 {
		t.Errorf("Compression: got %v", conf.Producer.Compression)
	}
	if _, ok := conf.Producer.Partitioner("t").(*stickyPartitioner); !ok {
		t.Error("expected sticky partitioner from env")
	}
}
