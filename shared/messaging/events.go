package messaging

import (
	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"
)

const (
	FindAvailableDriversQueue       = "find_available_drivers"
	DriverCmdTripRequestQueue       = "driver_cmd_trip_request"
	DriverCmdTripResponseQueue      = "driver_cmd_trip_response"
	NotifyDriverNoDriversFoundQueue = "notify_driver_no_drivers_found"
	NotifyDriverAssignedQueue       = "notify_driver_assigned_queue"
)

type TripCreatedEvent struct {
	Trip *pb.Trip `json:"trip"`
}

type DriveTripResponseData struct {
	Driver  *pbd.Driver `json:"driver"`
	TripID  string      `json:"tripID"`
	RiderID string      `json:"riderID"`
}
