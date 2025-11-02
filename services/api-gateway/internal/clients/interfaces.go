package clients

import (
	"context"
	driverPb "ride-sharing/shared/proto/driver"
	tripPb "ride-sharing/shared/proto/trip"
)

type TripServiceClient interface {
	PreviewTrip(ctx context.Context, previewTripRequest *tripPb.PreviewTripRequest) (*tripPb.PreviewTripResponse, error)
	CreateTrip(ctx context.Context, createTripRequest *tripPb.CreateTripRequest) (*tripPb.CreateTripResponse, error)
	Close()
}

type DriverServiceClient interface {
	RegisterDriver(ctx context.Context, registerDriverRequest *driverPb.RegisterDriverRequest) (*driverPb.RegisterDriverResponse, error)
	UnRegisterDriver(ctx context.Context, unRegisterDriverRequest *driverPb.RegisterDriverRequest) (*driverPb.RegisterDriverResponse, error)
	Close()
}
