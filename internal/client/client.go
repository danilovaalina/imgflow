package client

import (
	"context"
	"encoding/json"

	"imgflow/internal/model"

	"github.com/cockroachdb/errors"
	kafkago "github.com/segmentio/kafka-go"
)

type Producer interface {
	Publish(ctx context.Context, msg kafkago.Message) error
}

type Publisher struct {
	producer Producer // наш универсальный продюсер байтов
}

func NewPublisher(producer Producer) *Publisher {
	return &Publisher{producer: producer}
}

func (a *Publisher) Publish(ctx context.Context, task model.ImageTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return errors.WithStack(err)
	}

	return a.producer.Publish(ctx, kafkago.Message{
		Key:   []byte(task.ID.String()),
		Value: data,
	})
}

type ImageService interface {
	Process(ctx context.Context, task model.ImageTask) error
}

type Subscriber struct {
	svc ImageService
}

func NewSubscriber(svc ImageService) *Subscriber {
	return &Subscriber{svc: svc}
}

func (s *Subscriber) Handle(ctx context.Context, msg kafkago.Message) error {
	var task model.ImageTask
	if err := json.Unmarshal(msg.Value, &task); err != nil {
		return errors.WithStack(err)
	}

	return s.svc.Process(ctx, task)
}
