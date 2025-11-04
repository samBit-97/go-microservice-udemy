package websocket

import (
	"context"
	"encoding/json"
	"log"
	"ride-sharing/services/api-gateway/internal/clients"
	"ride-sharing/services/api-gateway/internal/websocket"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type WebSocketHandler struct {
	connManager   *websocket.ConnectionManager
	upgrader      *websocket.WebSocketUpgrader
	driverClient  clients.DriverServiceClient
	messageBroker messaging.MessageBroker
}

func NewWebSocketHandler(
	connManager *websocket.ConnectionManager,
	upgrader *websocket.WebSocketUpgrader,
	driverClient clients.DriverServiceClient,
	messageBroker messaging.MessageBroker) *WebSocketHandler {
	return &WebSocketHandler{
		connManager:   connManager,
		upgrader:      upgrader,
		driverClient:  driverClient,
		messageBroker: messageBroker,
	}
}

// createMessageHandler creates a RabbitMQ message handler that forwards messages to WebSocket connections
// This is used for consuming from RabbitMQ and forwarding to connected users (rider or driver)
func (h *WebSocketHandler) createMessageHandler(userType string) messaging.MessageHandler {
	return func(ctx context.Context, delivery amqp091.Delivery) error {
		var amqpMsg contracts.AmqpMessage
		if err := json.Unmarshal(delivery.Body, &amqpMsg); err != nil {
			log.Printf("Failed to unmarshal AMQP message for %s: %v, body: %s", userType, err, string(delivery.Body))
			// Don't requeue malformed messages - they'll never succeed
			return nil
		}

		userID := amqpMsg.OwnerID
		var payload any
		if amqpMsg.Data != nil {
			if err := json.Unmarshal(amqpMsg.Data, &payload); err != nil {
				log.Printf("Failed to unmarshal payload for %s %s: %v", userType, userID, err)
				// Don't requeue malformed messages
				return nil
			}
		}

		wsMsg := contracts.WSMessage{
			Type: delivery.RoutingKey,
			Data: payload,
		}

		// If sending fails (e.g., user not connected), log but don't requeue
		if err := h.connManager.SendMessage(userID, wsMsg); err != nil {
			log.Printf("Failed to send message to %s %s: %v", userType, userID, err)
			// User might not be connected yet, but message was valid - don't requeue
			return nil
		}

		log.Printf("Successfully forwarded message to %s %s: %s", userType, userID, delivery.RoutingKey)
		return nil
	}
}
