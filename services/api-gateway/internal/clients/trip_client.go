package clients

import (
	"context"
	"fmt"
	"log"
	"os"
	pb "ride-sharing/shared/proto/trip"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type tripServiceClient struct {
	client pb.TripServiceClient
	conn   *grpc.ClientConn
}

func NewTripServiceClient() (TripServiceClient, error) {
	tripServiceURL := os.Getenv("TRIP_SERVICE_URL")
	if tripServiceURL == "" {
		tripServiceURL = "trip-service:9093"
	}

	conn, err := grpc.NewClient(tripServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewTripServiceClient(conn)

	return &tripServiceClient{
		client: client,
		conn:   conn,
	}, nil
}

// CreateTrip implements TripServiceClient.
func (c *tripServiceClient) Close() {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			log.Printf("error closing the connection: %v", err)
			return
		}
	}
}

// CreateTrip implements TripServiceClient.
func (c *tripServiceClient) CreateTrip(ctx context.Context, createTripRequest *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	resp, err := c.client.CreateTrip(ctx, createTripRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create trip: %w", err)
	}

	return resp, nil
}

// PreviewTrip implements TripServiceClient.
func (c *tripServiceClient) PreviewTrip(ctx context.Context, previewTripRequest *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {
	resp, err := c.client.PreviewTrip(ctx, previewTripRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to preview trip: %w", err)
	}

	return resp, nil
}
