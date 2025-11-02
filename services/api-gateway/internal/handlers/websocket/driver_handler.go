package websocket

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"ride-sharing/shared/contracts"
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

	h.handleDriverMessages(conn, userID)
}

func (h *WebSocketHandler) handleDriverMessages(conn *websocket.Conn, userID string) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message from driver %s: %v", userID, err)
			break
		}
		log.Printf("Recieved message from driver %s: %v", userID, msg)
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
