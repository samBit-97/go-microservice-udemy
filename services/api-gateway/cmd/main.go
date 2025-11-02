package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-sharing/services/api-gateway/internal/clients"
	httpHandlers "ride-sharing/services/api-gateway/internal/handlers/http"
	wsHandlers "ride-sharing/services/api-gateway/internal/handlers/websocket"
	"ride-sharing/services/api-gateway/internal/websocket"
	"ride-sharing/shared/env"
)

var (
	httpAddr = env.GetString("HTTP_ADDR", ":8081")
)

func main() {
	log.Println("Starting API Gateway")
	wsUpgrader := websocket.NewWebSocketUpgrader()
	connManager := websocket.NewConnectionManager()

	tripClient, err := clients.NewTripServiceClient()
	if err != nil {
		log.Fatalf("Failed to create trip client: %v", err)
	}

	driverClient, err := clients.NewDriverServiceClient()
	if err != nil {
		log.Fatalf("Failed to create driver client: %v", err)
	}

	tripHandler := httpHandlers.NewTripHandler(tripClient)
	wsHandler := wsHandlers.NewWebSocketHandler(connManager, wsUpgrader, driverClient)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /trip/preview", httpHandlers.EnableCORS(tripHandler.HandleTripPreview))
	mux.HandleFunc("POST /trip/start", httpHandlers.EnableCORS(tripHandler.HandleCreateTrip))
	mux.HandleFunc("/ws/drivers", wsHandler.HandleDriverConnection)
	mux.HandleFunc("/ws/riders", wsHandler.HandleRiderConnection)

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("Server listening on port: %v", httpAddr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Printf("Error starting the server: %v", err)

	case sig := <-shutdown:
		log.Printf("Server is shutting down due to : %v signal", sig)

		// Close gRPC connections
		tripClient.Close()
		driverClient.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server could not shutdown gracefully: %v", err)
			server.Close()
		}

	}
}
