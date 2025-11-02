package websocket

import (
	"ride-sharing/services/api-gateway/internal/clients"
	"ride-sharing/services/api-gateway/internal/websocket"
)

type WebSocketHandler struct {
	connManager  *websocket.ConnectionManager
	upgrader     *websocket.WebSocketUpgrader
	driverClient clients.DriverServiceClient
}

func NewWebSocketHandler(
	connManager *websocket.ConnectionManager,
	upgrader *websocket.WebSocketUpgrader,
	driverClient clients.DriverServiceClient) *WebSocketHandler {
	return &WebSocketHandler{
		connManager:  connManager,
		upgrader:     upgrader,
		driverClient: driverClient,
	}
}
