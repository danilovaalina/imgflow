package kafka

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			Topic:       topic,
			GroupID:     groupID,
			MinBytes:    10e3, // 10KB
			MaxBytes:    10e6, // 10MB
			StartOffset: kafka.FirstOffset,
		}),
	}
}

type HandlerFunc func(ctx context.Context, msg kafka.Message) error

func (c *Consumer) Start(ctx context.Context, handler HandlerFunc) {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Error().Err(err).Send()
			continue
		}

		if err = handler(ctx, msg); err != nil {
			log.Error().Err(err).Int64("offset", msg.Offset).Int("partition", msg.Partition).Send()
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
