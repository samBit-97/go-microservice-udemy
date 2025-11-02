package websocket

import (
	"log"
	"net/http"
)

func (h *WebSocketHandler) HandleRiderConnection(w http.ResponseWriter, r *http.Request) {
	userID, err := validateUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r)
	if err != nil {
		log.Printf("Websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	h.connManager.Add(userID, conn)
	defer h.connManager.Remove(userID)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}
		log.Printf("Received message from rider %s: %s", userID, string(msg))
	}
}
