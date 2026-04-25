package ws

import (
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"

	"ws-service/internal/model"
)

// Hub хранит активные websocket-подключения пользователей и выполняет рассылку.
type Hub struct {
	mu          sync.RWMutex
	connections map[string]*websocket.Conn
	logger      *slog.Logger
}

// NewHub создает пустой Hub подключений.
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		connections: make(map[string]*websocket.Conn),
		logger:      logger,
	}
}

// Register регистрирует активное подключение пользователя, закрывая предыдущее при наличии.
func (h *Hub) Register(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if oldConn, ok := h.connections[userID]; ok {
		oldConn.Close()
	}
	h.connections[userID] = conn
}

// Unregister удаляет подключение пользователя из реестра.
func (h *Hub) Unregister(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.connections, userID)
}

// Broadcast отправляет сообщение всем переданным пользователям, если они онлайн.
func (h *Hub) Broadcast(userIDs []string, msg model.ChatMessage) {
	conns := make(map[string]*websocket.Conn)

	h.mu.RLock()
	for _, userID := range userIDs {
		conn, ok := h.connections[userID]
		if ok {
			conns[userID] = conn
		}
	}
	h.mu.RUnlock()

	for userID, conn := range conns {
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
