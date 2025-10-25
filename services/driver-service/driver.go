package main

import (
	"context"
	"ride-sharing/shared/messaging"
)

type TripEventConsumer interface {
	ConsumeTripCreated(ctx context.Context, queue string, handler messaging.MessageHandler) error
}
