package grpc

import (
	"context"
	"errors"
	"log"
	"ride-sharing/services/driver-service/internal/domain"
	pb "ride-sharing/shared/proto/driver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type driverHandler struct {
	pb.UnimplementedDriverServiceServer
	service domain.DriverService
}

// NewDriverHandler creates and registers a new driver gRPC handler
func NewDriverHandler(s *grpc.Server, service domain.DriverService) {
	handler := &driverHandler{
		service: service,
	}

	pb.RegisterDriverServiceServer(s, handler)
}

func (h *driverHandler) RegisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	driver, err := h.service.RegisterDriver(req.GetDriverID(), req.GetPackageSlug())
	if err != nil {
		log.Printf("Failed to register driver: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to register driver: %v", err)
	}

	return &pb.RegisterDriverResponse{
		Driver: driver,
	}, nil
}

func (h *driverHandler) UnregisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	err := h.service.UnregisterDriver(req.GetDriverID())
	if err != nil {
		// Driver not found is OK for idempotent operation
		if errors.Is(err, domain.ErrDriverNotFound) {
			return &pb.RegisterDriverResponse{
				Driver: &pb.Driver{
					Id: req.GetDriverID(),
				},
			}, nil
		}

		log.Printf("Failed to unregister driver: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to unregister driver: %v", err)
	}

	return &pb.RegisterDriverResponse{
		Driver: &pb.Driver{
			Id: req.GetDriverID(),
		},
	}, nil
}
