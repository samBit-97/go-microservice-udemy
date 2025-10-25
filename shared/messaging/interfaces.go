package messaging

import (
	"context"
	"ride-sharing/shared/contracts"

	"github.com/rabbitmq/amqp091-go"
)

// MessageHandler processes incoming messages
type MessageHandler func(ctx context.Context, msg amqp091.Delivery) error

// MessageBroker defines the contract for message publishing and subscription
type MessageBroker interface {
	Publish(ctx context.Context, routingKey string, msg contracts.AmqpMessage) error
	Consume(ctx context.Context, queue string, handler MessageHandler) error
	HealthCheck(ctx context.Context) error
	Close() error
}

// Config holds broker configuration
type Config struct {
	URI               string
	MaxRetries        int
	RetryDelay        string // Duration string like "1s", "100ms"
	ConnectionTimeout string // Duration string like "10s"
	PublishTimeout    string // Duration string like "5s"
}
