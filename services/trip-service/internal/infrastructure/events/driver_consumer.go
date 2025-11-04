package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pb "ride-sharing/shared/proto/driver"

	"github.com/rabbitmq/amqp091-go"
)

// driverConsumer implements domain.TripService
type driverConsumer struct {
	messageBroker messaging.MessageBroker
	service       domain.TripService
}

// NewDriverConsumer creates a new driver event consumer
// Following Dependency Inversion Principle (DIP):
// - Depends on domain.TripService interface, not concrete implementation
// - Depends on messaging.MessageBroker interface
func NewDriverConsumer(messageBroker messaging.MessageBroker, service domain.TripService) *driverConsumer {
	return &driverConsumer{
		messageBroker: messageBroker,
		service:       service,
	}
}

// ConsumeTripCreated starts consuming trip created events from the queue
func (c *driverConsumer) ConsumeDriverResponse(ctx context.Context, queue string, handler messaging.MessageHandler) error {
	// If no handler provided, use the default handleTripCreated
	if handler == nil {
		handler = c.handleDriverResponse
	}
	return c.messageBroker.Consume(ctx, queue, handler)
}

// handleTripCreated processes incoming trip created messages
func (c *driverConsumer) handleDriverResponse(ctx context.Context, delivery amqp091.Delivery) error {
	var msg contracts.AmqpMessage
	// Convert amqp.Delivery to contracts.AmqpMessage
	err := json.Unmarshal(delivery.Body, &msg)
	if err != nil {
		log.Printf("failed to unmarshal message: %v", err)
		return err
	}

	var payload messaging.DriveTripResponseData
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		log.Printf("failed to unmarshal message: %v", err)
		return err
	}

	log.Printf("Driver response recieved message: %+v", payload)

	switch delivery.RoutingKey {
	case contracts.DriverCmdTripAccept:
		if err := c.handleTripAccepted(ctx, payload.TripID, payload.Driver); err != nil {
			log.Printf("Failed to handle the trip accept %v", err)
			return err
		}
	case contracts.DriverCmdTripDecline:
		if err := c.handleTripDeclined(ctx, payload.TripID, payload.RiderID); err != nil {
			log.Printf("Failed to handle trip declined: %v", err)
			return err
		}
		return nil
	}

	return nil
}

func (c *driverConsumer) handleTripDeclined(ctx context.Context, tripID, riderID string) error {
	trip, err := c.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	newPayload := messaging.TripCreatedEvent{
		Trip: trip.ToProto(),
	}

	mashalledPayload, err := json.Marshal(newPayload)
	if err != nil {
		return err
	}

	if err := c.messageBroker.Publish(ctx, contracts.TripEventDriverNotInterested, contracts.AmqpMessage{
		OwnerID: riderID,
		Data:    mashalledPayload,
	}); err != nil {
		return err
	}

	return nil
}

func (c *driverConsumer) handleTripAccepted(ctx context.Context, tripID string, driver *pb.Driver) error {
	trip, err := c.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	if trip == nil {
		return fmt.Errorf("trip not found %s", tripID)
	}

	if err := c.service.UpdateTrip(ctx, tripID, "accepted", driver); err != nil {
		log.Printf("Failed to update trip: %v", err)
		return err
	}

	trip, err = c.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	marshalledTrip, err := json.Marshal(trip)
	if err != nil {
		return nil
	}

	//Notify the rider that a driver has been assigned
	if err := c.messageBroker.Publish(ctx, contracts.TripEventDriverAssigned, contracts.AmqpMessage{
		OwnerID: trip.UserID,
		Data:    marshalledTrip,
	}); err != nil {
		return err
	}

	return nil
}
