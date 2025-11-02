package clients

import (
	"context"
	"fmt"
	"log"
	"os"
	pb "ride-sharing/shared/proto/driver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type driverServiceClient struct {
	client pb.DriverServiceClient
	conn   *grpc.ClientConn
}

func NewDriverServiceClient() (DriverServiceClient, error) {
	driverServiceURL := os.Getenv("DRIVER_SERVICE_URL")
	if driverServiceURL == "" {
		driverServiceURL = "driver-service:9092"
	}

	conn, err := grpc.NewClient(driverServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewDriverServiceClient(conn)

	return &driverServiceClient{
		client: client,
		conn:   conn,
	}, nil
}

// RegisterDriver implements DriverServiceClient.
func (c *driverServiceClient) Close() {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			log.Printf("error closing connection: %v", err)
			return
		}
	}
}

// RegisterDriver implements DriverServiceClient.
func (c *driverServiceClient) RegisterDriver(ctx context.Context, registerDriverRequest *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	resp, err := c.client.RegisterDriver(ctx, registerDriverRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to register driver: %w", err)
	}

	return resp, nil
}

// UnRegisterDriver implements DriverServiceClient.
func (c *driverServiceClient) UnRegisterDriver(ctx context.Context, unRegisterDriverRequest *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	resp, err := c.client.UnregisterDriver(ctx, unRegisterDriverRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to unregister driver: %w", err)
	}

	return resp, nil
}
