package testing

import (
	"github.com/Shopify/sarama"
)

type Consumer struct {
	consumer sarama.Consumer
}

func NewConsumer(brokerAddress string) (*Consumer, error) {

	config := sarama.NewConfig()
	config.ClientID = "go-kafka-consumer"
	config.Consumer.Return.Errors = true

	brokers := []string{brokerAddress}

	// Create new consumer
	consumer, err := sarama.NewConsumer(brokers, config)

	return &Consumer{
		consumer: consumer,
	}, err
}

func (c *Consumer) Clean() error {
	return c.consumer.Close()
}

func (c *Consumer) Consume(topic string) (chan *sarama.ConsumerMessage, chan *sarama.ConsumerError) {
	return consume(topic, c.consumer)
}

func consume(topic string, master sarama.Consumer) (chan *sarama.ConsumerMessage, chan *sarama.ConsumerError) {
	messages := make(chan *sarama.ConsumerMessage)
	errors := make(chan *sarama.ConsumerError)
	partitions, _ := master.Partitions(topic)
	// only consume from partition[0]
	consumer, _ := master.ConsumePartition(topic, partitions[0], sarama.OffsetOldest)

	go func(consumer sarama.PartitionConsumer) {
		for {
			select {
			case consumerError := <-consumer.Errors():
				errors <- consumerError

			case msg := <-consumer.Messages():
				messages <- msg
			}
		}
	}(consumer)

	return messages, errors
}
