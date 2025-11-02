package messaging

import (
	"errors"
	"log"
	"net/http"
	"ride-sharing/shared/contracts"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	ErrConnectionNotFound = errors.New("connection not found")
)

type connWrapper struct {
	conn  *websocket.Conn
	mutex sync.Mutex
}

type ConnectionManager struct {
	connections map[string]*connWrapper
	mutex       sync.RWMutex
}

// TODO: REFACTOR - Extract WebSocketUpgrader into separate component (SRP violation)
// WHY: ConnectionManager has two separate responsibilities:
//   1. Managing the mapping of connections (storage/retrieval)
//   2. Handling HTTP-specific WebSocket protocol upgrades
// - These are independent concerns that can change for different reasons
// - Violates Single Responsibility Principle
// - Makes it impossible to customize upgrade behavior (CORS, compression, buffer sizes)
// - Mixing infrastructure concerns (HTTP) with business logic (connection tracking)
// ACTION: Extract WebSocketUpgrader into separate component, inject it into handlers
var upgrader = websocket.Upgrader{
	// TODO: REFACTOR - Remove hardcoded CheckOrigin security bypass (OCP + Security violation)
	// WHY: Hardcoding return true:
	//   - SECURITY RISK: Allows any origin to establish WebSocket connections (CORS bypass)
	//   - Not extensible - cannot change validation per environment (dev vs production)
	//   - Cannot implement whitelist, JWT, or other validation strategies
	//   - Violates Open/Closed Principle (closed for extension)
	// ACTION: Extract into OriginValidator interface, support multiple strategies:
	//   - WhitelistOriginValidator (list of allowed origins)
	//   - AllowAllOriginValidator (dev/test only)
	//   - JWTOriginValidator (validate with JWT token)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*connWrapper),
	}
}

// TODO: REFACTOR - Move Upgrade method to WebSocketUpgrader (SRP violation)
// WHY: ConnectionManager shouldn't be responsible for HTTP WebSocket protocol upgrade:
//   - HTTP concerns don't belong in connection manager
//   - This method is HTTP-specific (depends on http.ResponseWriter, *http.Request)
//   - Makes it harder to test ConnectionManager in isolation
//   - Cannot reuse ConnectionManager without HTTP dependencies
// ACTION: Extract into WebSocketUpgrader struct with its own constructor and methods
func (cm *ConnectionManager) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (cm *ConnectionManager) Add(id string, conn *websocket.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.connections[id] = &connWrapper{
		conn:  conn,
		mutex: sync.Mutex{},
	}

	log.Printf("Added connection for user %s", id)
}

func (cm *ConnectionManager) Remove(id string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	delete(cm.connections, id)
}

func (cm *ConnectionManager) Get(id string) (*websocket.Conn, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	wrapper, exists := cm.connections[id]
	if !exists {
		return nil, false
	}
	return wrapper.conn, true
}

func (cm *ConnectionManager) SendMessage(id string, message contracts.WSMessage) error {
	cm.mutex.RLock()
	wrapper, exists := cm.connections[id]
	cm.mutex.RUnlock()

	if !exists {
		return ErrConnectionNotFound
	}

	wrapper.mutex.Lock()
	defer wrapper.mutex.Unlock()

	return wrapper.conn.WriteJSON(message)
}
