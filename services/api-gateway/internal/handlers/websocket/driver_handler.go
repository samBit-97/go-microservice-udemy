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

	// Use the common message handler to forward RabbitMQ messages to driver's WebSocket
	handler := h.createMessageHandler("driver")
	for _, q := range queues {
		if err := h.messageBroker.Consume(ctx, q, handler); err != nil {
			log.Printf("Consumer error for queue %s: %v", q, err)
		}
	}

	h.handleDriverMessages(ctx, conn, userID)
}

func (h *WebSocketHandler) handleDriverMessages(ctx context.Context, conn *websocket.Conn, userID string) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message from driver %s: %v", userID, err)
			break
		}

		type driverMessage struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}

		var driverMsg driverMessage
		if err := json.Unmarshal(msg, &driverMsg); err != nil {
			log.Printf("Error unmarshalling driver message: %v", err)
			continue
		}

		switch driverMsg.Type {

		case contracts.DriverCmdLocation:
			continue

		case contracts.DriverCmdTripAccept, contracts.DriverCmdTripDecline:
			// Extract riderID from the message data to use as OwnerID
			var tripResponse struct {
				RiderID string `json:"riderID"`
			}
			if err := json.Unmarshal(driverMsg.Data, &tripResponse); err != nil {
				log.Printf("Error unmarshaling trip response data: %v", err)
				continue
			}

			if err := h.messageBroker.Publish(ctx, driverMsg.Type, contracts.AmqpMessage{
				OwnerID: tripResponse.RiderID, // Use rider's ID, not driver's ID
				Data:    driverMsg.Data,
			}); err != nil {
				log.Printf("Error publishing message to rabbitmq: %v", err)
			}

		default:
			log.Printf("Unknown message type: %v", driverMsg.Type)
		}
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
