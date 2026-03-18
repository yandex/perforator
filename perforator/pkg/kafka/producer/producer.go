package producer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type Header = kafkago.Header

type Producer interface {
	Produce(ctx context.Context, partitionKey []byte, value []byte, headers ...Header) error
	Close() error
}

type KafkaProducer struct {
	w *kafkago.Writer
}

func NewKafkaProducer(l xlog.Logger, cfg *Config) (*KafkaProducer, error) {
	ctx := context.Background()
	w, err := NewKafkaWriter(ctx, l, cfg)
	if err != nil {
		return nil, err
	}
	return &KafkaProducer{w: w}, nil
}

// Produce publishes a message to Kafka.
// Messages with the same key are routed to the same partition.
// If the key is empty or nil, the message is routed by the round-robin algorithm.
func (p *KafkaProducer) Produce(ctx context.Context, partitionKey []byte, value []byte, headers ...Header) error {
	if p.w == nil {
		return errors.New("writer is nil")
	}
	return p.w.WriteMessages(ctx, kafkago.Message{Key: partitionKey, Value: value, Headers: headers})
}

func (p *KafkaProducer) Close() error {
	return p.w.Close()
}

/////////////////////////////////////////////////////////////////////////////////////////

type JSONProducer[T any] struct {
	base Producer
}

func NewJSONProducer[T any](base Producer) *JSONProducer[T] {
	return &JSONProducer[T]{base: base}
}

func (j *JSONProducer[T]) Publish(ctx context.Context, partitionKey []byte, msg *T, headers ...Header) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal json before publish: %w", err)
	}
	return j.base.Produce(ctx, partitionKey, b, headers...)
}

func (j *JSONProducer[T]) Close() error {
	return j.base.Close()
}
