package messaging

import (
	"context"
	"encoding/json"
	"log"
	"ride-sharing/services/driver-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

// tripConsumer implements domain.TripEventConsumer
type tripConsumer struct {
	messageBroker messaging.MessageBroker
	service       domain.DriverService
}

// NewTripConsumer creates a new trip event consumer
// Following Dependency Inversion Principle (DIP):
// - Depends on domain.DriverService interface, not concrete implementation
// - Depends on messaging.MessageBroker interface
func NewTripConsumer(messageBroker messaging.MessageBroker, service domain.DriverService) domain.TripEventConsumer {
	return &tripConsumer{
		messageBroker: messageBroker,
		service:       service,
	}
}

// ConsumeTripCreated starts consuming trip created events from the queue
func (c *tripConsumer) ConsumeTripCreated(ctx context.Context, queue string, handler messaging.MessageHandler) error {
	// If no handler provided, use the default handleTripCreated
	if handler == nil {
		handler = c.handleTripCreated
	}
	return c.messageBroker.Consume(ctx, queue, handler)
}

// handleTripCreated processes incoming trip created messages
func (c *tripConsumer) handleTripCreated(ctx context.Context, delivery amqp091.Delivery) error {
	var msg contracts.AmqpMessage
	// Convert amqp.Delivery to contracts.AmqpMessage
	err := json.Unmarshal(delivery.Body, &msg)
	if err != nil {
		log.Printf("failed to unmarshal message: %v", err)
		return err
	}

	var tripEvent messaging.TripCreatedEvent
	if err := json.Unmarshal(msg.Data, &tripEvent); err != nil {
		log.Printf("failed to unmarshal message: %v", err)
		return err
	}

	log.Printf("Driver recieved message: %+v", tripEvent)

	switch delivery.RoutingKey {
	case contracts.TripEventCreated, contracts.TripEventDriverNotInterested:
		return c.handleTripEventCreated(ctx, tripEvent)
	}

	return nil
}

// handleTripEventCreated processes trip created events and publishes appropriate responses
func (c *tripConsumer) handleTripEventCreated(ctx context.Context, tripEvent messaging.TripCreatedEvent) error {
	driverID, err := c.service.FindAndNotifyDrivers(ctx, tripEvent)
	if err != nil {
		log.Print(err)
		return c.publishDriverNotFoundEvent(ctx, tripEvent)
	}

	log.Printf("found driver %s for trip %s", driverID, tripEvent.Trip.Id)
	return c.publishDriverFoundEvent(ctx, driverID, tripEvent)
}

// publishDriverNotFoundEvent publishes an event when no drivers are available
func (c *tripConsumer) publishDriverNotFoundEvent(ctx context.Context, tripEvent messaging.TripCreatedEvent) error {
	log.Printf("publishing message with routing key: %v", contracts.TripEventNoDriversFound)
	return c.messageBroker.Publish(ctx,
		contracts.TripEventNoDriversFound,
		contracts.AmqpMessage{
			OwnerID: tripEvent.Trip.UserID,
		})
}

// publishDriverFoundEvent publishes an event when a driver is found and assigned
func (c *tripConsumer) publishDriverFoundEvent(ctx context.Context, driverID string, tripEvent messaging.TripCreatedEvent) error {
	log.Printf("publishing message with routing key: %v for driver: %s", contracts.DriverCmdTripRequest, driverID)
	marshalledEvent, err := json.Marshal(tripEvent)
	if err != nil {
		return err
	}

	return c.messageBroker.Publish(ctx, contracts.DriverCmdTripRequest, contracts.AmqpMessage{
		OwnerID: driverID,
		Data:    marshalledEvent,
	})
}
