package ws

import (
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"

	"ws-service/internal/model"
)

type Hub struct {
	mu          sync.RWMutex
	connections map[string]*websocket.Conn
	logger      *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		connections: make(map[string]*websocket.Conn),
		logger:      logger,
	}
}

func (h *Hub) Register(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[userID] = conn
}

func (h *Hub) Unregister(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.connections, userID)
}

func (h *Hub) Broadcast(userIDs []string, msg model.ChatMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, userID := range userIDs {
		conn, ok := h.connections[userID]
		if !ok {
			continue
		}

		if err := conn.WriteJSON(msg); err != nil {
			h.logger.Error("failed to write websocket message",
				"component", "ws.hub",
				"operation", "broadcast.write_json",
				"user_id", userID,
				"chat_id", msg.ChatID,
				"error", err,
			)
		}
	}
}
