package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ride-sharing/shared/contracts"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	TripExchange = "trip"
)

type rabbitmqBroker struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQ(uri string) (*rabbitmqBroker, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	rmq := &rabbitmqBroker{
		conn:    conn,
		channel: ch,
	}

	if err := rmq.setupExchangeAndQueues(); err != nil {
		rmq.Close()
		return nil, fmt.Errorf("failed to setup exchanges and queues: %w", err)
	}

	return rmq, nil
}

func (r *rabbitmqBroker) setupExchangeAndQueues() error {
	err := r.channel.ExchangeDeclare(
		TripExchange,
		"topic",
		true,
		false,
		false,
		false,
		nil)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %s : %w", TripExchange, err)
	}

	// Queue for driver-service to find available drivers for trips
	if err := r.declareAndBindQueue(
		FindAvailableDriversQueue,
		[]string{
			contracts.TripEventCreated, contracts.TripEventDriverNotInterested,
		},
		TripExchange); err != nil {
		return err
	}

	// Queue for API Gateway to forward trip requests to drivers via WebSocket
	if err := r.declareAndBindQueue(
		DriverCmdTripRequestQueue,
		[]string{
			contracts.DriverCmdTripRequest, // Driver found and assigned to trip
		},
		TripExchange); err != nil {
		return err
	}

	// Queue for API Gateway to forward driver responses to riders via WebSocket
	if err := r.declareAndBindQueue(
		DriverCmdTripResponseQueue,
		[]string{
			contracts.DriverCmdTripAccept,  // Driver accepted trip
			contracts.DriverCmdTripDecline, // Driver declined trip
		},
		TripExchange); err != nil {
		return err
	}

	// Queue for API Gateway to notify riders when no drivers are available
	if err := r.declareAndBindQueue(
		NotifyDriverNoDriversFoundQueue,
		[]string{
			contracts.TripEventNoDriversFound,
		},
		TripExchange); err != nil {
		return err
	}

	if err := r.declareAndBindQueue(
		NotifyDriverAssignedQueue,
		[]string{
			contracts.TripEventDriverAssigned,
		},
		TripExchange); err != nil {
		return err
	}

	return nil
}

func (r *rabbitmqBroker) declareAndBindQueue(queueName string, messageTypes []string, exchange string) error {
	q, err := r.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return err
	}

	for _, msg := range messageTypes {
		if err := r.channel.QueueBind(
			q.Name,
			msg,
			exchange,
			false,
			nil); err != nil {
			return fmt.Errorf("failed to bind queue: %s: %v", queueName, err)
		}
	}

	return nil
}

func (r *rabbitmqBroker) Publish(ctx context.Context, routingKey string, message contracts.AmqpMessage) error {

	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	return r.channel.PublishWithContext(ctx,
		TripExchange, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "text/plain",
			Body:         jsonMsg,
			DeliveryMode: amqp.Persistent,
		})
}

func (r *rabbitmqBroker) HealthCheck(ctx context.Context) error {
	if r.conn == nil {
		return fmt.Errorf("connection is nil")
	}
	if r.conn.IsClosed() {
		return fmt.Errorf("connection is closed")
	}
	if r.channel == nil {
		return fmt.Errorf("channel is nil")
	}
	if r.channel.IsClosed() {
		return fmt.Errorf("channel is closed")
	}
	return nil
}

func (r *rabbitmqBroker) Close() error {
	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			return fmt.Errorf("failed to close channel: %w", err)
		}
	}
	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}
	return nil
}

func (r *rabbitmqBroker) Consume(ctx context.Context, queue string, handler MessageHandler) error {
	err := r.channel.Qos(1, 0, false)
	if err != nil {
		return fmt.Errorf("failed to set Qos: %w", err)
	}

	deliveries, err := r.channel.Consume(
		queue,
		"",    // consumer tag
		false, // auto-ack disabled for manual acknowledgment
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil)

	if err != nil {
		return fmt.Errorf("failed to register consumer on queue %s: %w", queue, err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("Consumer stopped for queue %s: context cancelled", queue)
				return
			case delivery, ok := <-deliveries:
				if !ok {
					log.Printf("Consumer stopped for queue %s: channel closed", queue)
					return
				}

				// Call handler
				if err := handler(ctx, delivery); err != nil {
					log.Printf("failed to handle message %v", err)
					delivery.Nack(false, true) // Requeue on handler error
					continue
				}

				// Acknowledge successful processing
				delivery.Ack(false)
			}
		}
	}()

	return nil
}
