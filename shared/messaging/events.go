package messaging

import (
	pb "ride-sharing/shared/proto/trip"
)

const (
	FindAvailableDriversQueue = "find_available_drivers"
	DriverCmdTripRequestQueue = "driver_cmd_trip_request"
)

type TripCreatedEvent struct {
	Trip *pb.Trip `json:"trip"`
}
