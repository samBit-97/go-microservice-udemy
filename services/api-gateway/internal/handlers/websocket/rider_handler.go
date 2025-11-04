package websocket

import (
	"log"
	"net/http"
	"ride-sharing/shared/messaging"
)

func (h *WebSocketHandler) HandleRiderConnection(w http.ResponseWriter, r *http.Request) {
	userID, err := validateUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	conn, err := h.upgrader.Upgrade(w, r)
	if err != nil {
		log.Printf("Websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	h.connManager.Add(userID, conn)
	defer h.connManager.Remove(userID)

	queues := []string{
		messaging.NotifyDriverNoDriversFoundQueue,
		messaging.NotifyDriverAssignedQueue,
	}

	// Use the common message handler to forward RabbitMQ messages to driver's WebSocket
	handler := h.createMessageHandler("rider")
	for _, q := range queues {
		if err := h.messageBroker.Consume(ctx, q, handler); err != nil {
			log.Printf("Consumer error for queue %s: %v", q, err)
		}
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}
		log.Printf("Received message from rider %s: %s", userID, string(msg))
	}
}
