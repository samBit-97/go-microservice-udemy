package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/services/trip-service/internal/infrastructure/events"
	"ride-sharing/services/trip-service/internal/infrastructure/grpc"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

var GrpcAddr = ":9093"

func main() {

	rabbitMqUri := env.GetString("RABBITMQ_URI", "")
	if rabbitMqUri == "" {
		log.Fatal("RABBITMQ_URI environment variable is required")
	}
	repo := repository.NewInmemRepository()
	svc := service.NewService(repo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cancel()
	}()

	rabbitMq, err := messaging.NewRabbitMQ(rabbitMqUri)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitMq.Close()

	log.Println("Starting Rabbit MQ connection")
	publisher := events.NewTripEventPublisher(rabbitMq)

	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpcserver.NewServer()
	grpc.NewGRPCHandler(grpcServer, svc, publisher)

	log.Printf("Starting gRPC server TripService on port: %v", lis.Addr().String())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("Failed to serve: %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	log.Print("Shutting down the server...")
	grpcServer.GracefulStop()
}
