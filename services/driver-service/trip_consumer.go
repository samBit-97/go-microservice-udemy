package main

import (
	"context"
	"encoding/json"
	"log"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type tripConsumer struct {
	messageBroker messaging.MessageBroker
	service       *Service
}

func NewTripConsumer(messageBroker messaging.MessageBroker, service *Service) *tripConsumer {
	return &tripConsumer{
		messageBroker: messageBroker,
		service:       service,
	}
}

func (c *tripConsumer) ConsumeTripCreated(ctx context.Context, queue string, handler messaging.MessageHandler) error {
	return c.messageBroker.Consume(ctx, queue, handler)
}

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

func (c *tripConsumer) handleTripEventCreated(ctx context.Context, tripEvent messaging.TripCreatedEvent) error {
	driverID, err := c.service.FindAndNotifyDrivers(ctx, tripEvent)
	if err != nil {
		log.Print(err)
		return c.publishDriverNotFoundEvent(ctx, tripEvent)
	}

	log.Printf("found driver %s for trip %s", driverID, tripEvent.Trip.Id)
	return c.publishDriverFoundEvent(ctx, tripEvent)
}

func (c *tripConsumer) publishDriverNotFoundEvent(ctx context.Context, tripEvent messaging.TripCreatedEvent) error {
	log.Printf("publishing message with routing key: %v", contracts.TripEventNoDriversFound)
	return c.messageBroker.Publish(ctx,
		contracts.TripEventNoDriversFound,
		contracts.AmqpMessage{
			OwnerID: tripEvent.Trip.UserID,
		})
}

func (c *tripConsumer) publishDriverFoundEvent(ctx context.Context, tripEvent messaging.TripCreatedEvent) error {
	log.Printf("publishing message with routing key: %v", contracts.DriverCmdRegister)
	marshalledEvent, err := json.Marshal(tripEvent)
	if err != nil {
		return err
	}

	return c.messageBroker.Publish(ctx, contracts.DriverCmdRegister, contracts.AmqpMessage{
		OwnerID: tripEvent.Trip.UserID,
		Data:    marshalledEvent,
	})
}
