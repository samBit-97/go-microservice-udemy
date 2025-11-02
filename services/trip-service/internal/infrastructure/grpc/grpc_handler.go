package grpc

import (
	"context"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCHandler struct {
	pb.UnimplementedTripServiceServer
	service   domain.TripService
	publisher domain.TripEventPublisher
}

func NewGRPCHandler(server *grpc.Server, service domain.TripService, publisher domain.TripEventPublisher) *gRPCHandler {
	handler := &gRPCHandler{
		service:   service,
		publisher: publisher,
	}

	pb.RegisterTripServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) PreviewTrip(ctx context.Context, req *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {
	pickup := req.GetStartLocation()
	destination := req.GetEndLocation()

	pickupCoordinate := &types.Coordinate{
		Latitude:  pickup.Latitude,
		Longitude: pickup.Longitude,
	}
	destinationCoordinate := &types.Coordinate{
		Latitude:  destination.Latitude,
		Longitude: destination.Longitude,
	}

	route, err := h.service.GetRoute(ctx, pickupCoordinate, destinationCoordinate)
	if err != nil {
		log.Println(err)
		return nil, status.Errorf(codes.Internal, "Failed to get route: %v", err)
	}

	userID := req.UserID

	estimatedFares := h.service.EstimatePackagesPriceWithRoute(route)
	fares, err := h.service.GenerateTripFares(ctx, estimatedFares, userID, route)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate ride fares: %v", err)
	}

	return &pb.PreviewTripResponse{
		Route:     route.ToProto(),
		RideFares: domain.ToRideFaresProto(fares),
	}, nil
}

// TODO: REFACTOR - Fix inconsistent state handling in CreateTrip (LSP violation)
// WHY: Current implementation violates Liskov Substitution Principle:
//   1. Trip is created and persisted to database
//   2. If publishing event fails, method returns error
//   3. BUT the trip was already created - client receives error but trip exists
//   4. Creates orphaned data and inconsistent distributed state
//   5. Client expects either: full success OR no side effects - current behavior violates this contract
//   6. If publisher is replaced with different implementation, behavioral expectations break
// RISKS:
//   - Trip created but driver never notified (trip assigned but no one comes)
//   - Customer charged but driver doesn't see the request
//   - Message queue failure causes silent data corruption
//   - No way to retry publishing without reprocessing
// ACTION: Implement Outbox Pattern or async publishing:
//   1. Store trip creation event in outbox table within same transaction
//   2. Return success immediately once trip is created
//   3. Background worker async publishes events with retry logic
//   4. Maintains eventual consistency without blocking client
//   5. Provides audit trail of all events
// See: https://martinfowler.com/articles/patterns-of-distributed-systems/outbox.html
func (h *gRPCHandler) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	fareID := req.GetRideFareID()
	userID := req.GetUserID()
	rideFare, err := h.service.GetAndValidateFare(ctx, fareID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to validate the fare: %v", err)
	}

	trip, err := h.service.CreateTrip(ctx, rideFare)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create the trip: %v", err)
	}

	// TODO: REFACTOR - Replace synchronous publish with async outbox pattern
	// WHY: Current synchronous publishing is problematic:
	//   - Blocks client waiting for message broker
	//   - No retry mechanism - single failure cascades to client error
	//   - No audit trail of events
	//   - Violates eventual consistency patterns for microservices
	// ACTION: Store event in outbox, background worker publishes asynchronously
	if err := h.publisher.PublishTripCreated(ctx, trip); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to publish the tripEvent: %v", err)
	}

	return &pb.CreateTripResponse{
		TripID: trip.ID.Hex(),
	}, nil
}
