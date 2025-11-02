package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/services/driver-service/internal/domain"
	grpcHandler "ride-sharing/services/driver-service/internal/infrastructure/grpc"
	messagingInfra "ride-sharing/services/driver-service/internal/infrastructure/messaging"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

const (
	defaultGrpcAddr = ":9092"
)

func main() {
	// Load environment variables
	rabbitMQuri := env.GetString("RABBITMQ_URI", "")
	if rabbitMQuri == "" {
		log.Fatal("RABBITMQ_URI environment variable is required")
	}

	grpcAddr := env.GetString("GRPC_ADDR", defaultGrpcAddr)

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Initialize infrastructure
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Initialize RabbitMQ connection
	log.Println("Connecting to RabbitMQ...")
	rabbitMq, err := messaging.NewRabbitMQ(rabbitMQuri)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitMq.Close()
	log.Println("RabbitMQ connection established")

	// Initialize domain service
	// Following Dependency Inversion Principle (DIP):
	// All components depend on interfaces, not concrete implementations
	driverService := domain.NewDriverService()

	// Initialize gRPC server and register handlers
	grpcServer := grpcserver.NewServer()
	grpcHandler.NewDriverHandler(grpcServer, driverService)

	// Initialize RabbitMQ consumer
	tripConsumer := messagingInfra.NewTripConsumer(rabbitMq, driverService)

	// Start RabbitMQ consumer in background
	go func() {
		log.Printf("Starting RabbitMQ consumer for queue: %s", messaging.FindAvailableDriversQueue)
		// Note: handleTripCreated is unexported, so we need to expose it via the interface
		// For now, we'll start the consumer directly - this will be handled in the consumer implementation
		if err := tripConsumer.ConsumeTripCreated(ctx, messaging.FindAvailableDriversQueue, nil); err != nil {
			log.Printf("Consumer error: %v", err)
			cancel()
		}
	}()

	// Start gRPC server in background
	go func() {
		log.Printf("Starting gRPC server DriverService on %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("Failed to serve: %v", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Print("Shutting down the server...")

	// Graceful shutdown
	grpcServer.GracefulStop()
	log.Print("Server stopped gracefully")
}
