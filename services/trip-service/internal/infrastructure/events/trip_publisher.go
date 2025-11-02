package events

import (
	"context"
	"encoding/json"
	"fmt"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
)

type tripEventPublisher struct {
	messageBroker messaging.MessageBroker
}

func NewTripEventPublisher(messageBroker messaging.MessageBroker) domain.TripEventPublisher {
	return &tripEventPublisher{
		messageBroker: messageBroker,
	}
}

func (p *tripEventPublisher) PublishTripCreated(ctx context.Context, trip *domain.TripModel) error {

	event := messaging.TripCreatedEvent{
		Trip: trip.ToProto(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal trip created event: %w", err)
	}

	msg := contracts.AmqpMessage{
		OwnerID: trip.UserID,
		Data:    payload,
	}

	return p.messageBroker.Publish(ctx, contracts.TripEventCreated, msg)
}
