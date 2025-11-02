package websocket

import (
	"ride-sharing/services/api-gateway/internal/clients"
	"ride-sharing/services/api-gateway/internal/websocket"
	"ride-sharing/shared/messaging"
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
