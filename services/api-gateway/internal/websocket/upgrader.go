package websocket

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type WebSocketUpgrader struct {
	upgrader websocket.Upgrader
}

func NewWebSocketUpgrader() *WebSocketUpgrader {
	return &WebSocketUpgrader{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true //TODO: Fix security issue - allows any origin
			},
		},
	}
}

func (u *WebSocketUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := u.upgrader.Upgrade(w, r, nil)
	return conn, err
}
