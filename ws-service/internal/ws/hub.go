package ws

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"ws-service/internal/model"
)

const hubCloseWriteDeadline = 5 * time.Second

// Hub хранит активные websocket-подключения пользователей и выполняет рассылку.
type Hub struct {
	mu          sync.RWMutex
	connections map[string]*websocket.Conn
	logger      *slog.Logger
	closed      bool
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
	if h.closed {
		_ = conn.Close()
		return
	}
	if oldConn, ok := h.connections[userID]; ok {
		oldConn.Close()
	}
	h.connections[userID] = conn
}

// Unregister удаляет подключение пользователя из реестра.
func (h *Hub) Unregister(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	delete(h.connections, userID)
}

// Close закрывает все WebSocket-подключения и запрещает новые регистрации (идемпотентно).
func (h *Hub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	snapshot := make([]*websocket.Conn, 0, len(h.connections))
	for _, c := range h.connections {
		snapshot = append(snapshot, c)
	}
	h.connections = make(map[string]*websocket.Conn)
	h.mu.Unlock()

	deadline := time.Now().Add(hubCloseWriteDeadline)
	for _, conn := range snapshot {
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server shutting down"),
			deadline,
		)
		_ = conn.Close()
	}
}

// Broadcast отправляет сообщение всем переданным пользователям, если они онлайн.
func (h *Hub) Broadcast(userIDs []string, msg model.ChatMessage) {
	conns := make(map[string]*websocket.Conn)

	h.mu.RLock()
	if h.closed {
		h.mu.RUnlock()
		return
	}
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
