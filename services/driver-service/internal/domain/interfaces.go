package domain

import (
	"context"
	"errors"
	"ride-sharing/shared/messaging"
	pb "ride-sharing/shared/proto/driver"
)

var (
	// ErrDriverNotFound is returned when a driver is not found
	ErrDriverNotFound = errors.New("driver not found")
)

// DriverService defines the contract for driver management operations
type DriverService interface {
	// RegisterDriver registers a new driver with the system
	RegisterDriver(driverID string, packageSlug string) (*pb.Driver, error)

	// UnregisterDriver removes a driver from the system
	UnregisterDriver(driverID string) error

	// ProcessTripCreatedEvent processes trip creation events
	ProcessTripCreatedEvent(ctx context.Context, tripID, userID string) error

	// FindAndNotifyDrivers finds available drivers and notifies them of a trip
	FindAndNotifyDrivers(ctx context.Context, tripEvent messaging.TripCreatedEvent) (string, error)
}

// TripEventConsumer defines the contract for consuming trip events
type TripEventConsumer interface {
	// ConsumeTripCreated starts consuming trip created events from the queue
	ConsumeTripCreated(ctx context.Context, queue string, handler messaging.MessageHandler) error
}
