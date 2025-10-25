package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

var GrpcAddr = ":9092"

func main() {
	rabbitMQuri := env.GetString("RABBITMQ_URI", "")
	if rabbitMQuri == "" {
		log.Fatal("RABBITMQ_URI environment variable is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cancel()
	}()

	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	rabbitMq, err := messaging.NewRabbitMQ(rabbitMQuri)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitMq.Close()

	log.Println("Starting Rabbit MQ connection")
	service := NewService()

	grpcServer := grpcserver.NewServer()
	NewGrpcHandler(grpcServer, service)

	consumer := NewTripConsumer(rabbitMq, service)

	go func() {
		if err := consumer.ConsumeTripCreated(ctx, messaging.FindAvailableDriversQueue, consumer.handleTripCreated); err != nil {
			log.Printf("Consumer error: %v", err)
			cancel()
		}
	}()

	log.Printf("Starting gRPC server DriverService on port: %v", lis.Addr().String())

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
