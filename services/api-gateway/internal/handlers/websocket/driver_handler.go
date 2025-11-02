package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pb "ride-sharing/shared/proto/driver"

	"github.com/gorilla/websocket"
	"github.com/rabbitmq/amqp091-go"
)

func (h *WebSocketHandler) HandleDriverConnection(w http.ResponseWriter, r *http.Request) {
	userID, packageSlug, err := validateDriverParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	ctx := r.Context()
	h.connManager.Add(userID, conn)

	defer func() {
		h.connManager.Remove(userID)
		h.unregisterDriver(ctx, userID, packageSlug)
		log.Printf("Driver unregistered: %s", userID)
	}()

	if err := h.registerDriver(ctx, userID, packageSlug); err != nil {
		log.Printf("Failed to register driver: %v", err)
		return
	}

	queues := []string{
		messaging.DriverCmdTripRequestQueue,
	}

	handler := h.createDriverMessageHandler()
	for _, q := range queues {
		if err := h.messageBroker.Consume(ctx, q, handler); err != nil {
			log.Printf("Consumer error for queue %s: %v", q, err)
		}
	}

	h.handleDriverMessages(conn, userID)
}

func (h *WebSocketHandler) handleDriverMessages(conn *websocket.Conn, userID string) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message from driver %s: %v", userID, err)
			break
		}
		log.Printf("Received message from driver %s: %s", userID, string(msg))
	}
}

func (h *WebSocketHandler) registerDriver(ctx context.Context, userID string, packageSlug string) error {
	resp, err := h.driverClient.RegisterDriver(ctx, &pb.RegisterDriverRequest{
		DriverID:    userID,
		PackageSlug: packageSlug,
	})

	if err != nil {
		return fmt.Errorf("failed to register driver: %w", err)
	}

	return h.connManager.SendMessage(userID, contracts.WSMessage{
		Type: contracts.DriverCmdRegister,
		Data: resp.Driver,
	})
}

func (h *WebSocketHandler) unregisterDriver(ctx context.Context, userID string, packageSlug string) {
	_, err := h.driverClient.UnRegisterDriver(ctx, &pb.RegisterDriverRequest{
		DriverID:    userID,
		PackageSlug: packageSlug,
	})
	if err != nil {
		log.Printf("Error unregistering driver %s: %v", userID, err)
	}
}

func (h *WebSocketHandler) createDriverMessageHandler() messaging.MessageHandler {
	return func(ctx context.Context, delivery amqp091.Delivery) error {
		var amqpMsg contracts.AmqpMessage
		if err := json.Unmarshal(delivery.Body, &amqpMsg); err != nil {
			log.Printf("Failed to unmarshal AMQP message: %v, body: %s", err, string(delivery.Body))
			// Don't requeue malformed messages - they'll never succeed
			return nil
		}

		userID := amqpMsg.OwnerID
		var payload any
		if amqpMsg.Data != nil {
			if err := json.Unmarshal(amqpMsg.Data, &payload); err != nil {
				log.Printf("Failed to unmarshal payload for user %s: %v", userID, err)
				// Don't requeue malformed messages
				return nil
			}
		}

		wsMsg := contracts.WSMessage{
			Type: delivery.RoutingKey,
			Data: payload,
		}

		// If sending fails (e.g., driver not connected), log but don't requeue
		if err := h.connManager.SendMessage(userID, wsMsg); err != nil {
			log.Printf("Failed to send message to driver %s: %v", userID, err)
			// Driver might not be connected yet, but message was valid - don't requeue
			return nil
		}

		log.Printf("Successfully forwarded message to driver %s: %s", userID, delivery.RoutingKey)
		return nil
	}
}
