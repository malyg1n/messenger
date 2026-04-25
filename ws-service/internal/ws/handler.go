package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"

	"ws-service/internal/broker"
	"ws-service/internal/model"
)

// Handler обслуживает websocket-подключения и отправляет сообщения в Kafka.
type Handler struct {
	producer broker.Producer
	hub      *Hub
	logger   *slog.Logger
	upgrader websocket.Upgrader
}

// NewHandler создает websocket-handler с доступом к Kafka producer и Hub.
func NewHandler(producer broker.Producer, hub *Hub, logger *slog.Logger) *Handler {
	return &Handler{
		producer: producer,
		hub:      hub,
		logger:   logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}
}

// HandleWS обрабатывает lifecycle websocket-сессии конкретного пользователя.
func (h *Handler) HandleWS(w http.ResponseWriter, r *http.Request) {
	// user_id — ключ маршрутизации в Hub: для пользователя держим одно активное подключение.
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		h.logger.Warn("missing required websocket query param",
			"component", "ws.handler",
			"operation", "handle_ws.validate_user_id",
		)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("failed to upgrade websocket connection",
			"component", "ws.handler",
			"operation", "handle_ws.upgrade",
			"user_id", userID,
			"error", err,
		)
		return
	}
	conn.SetReadLimit(16 * 1024) // 16KB
	defer conn.Close()

	h.hub.Register(userID, conn)
	defer h.hub.Unregister(userID)

	for {
		// Читаем сообщение клиента, валидируем и публикуем в Kafka.
		// Рассылка получателям происходит асинхронно через consumer + hub.
		_, payload, err := conn.ReadMessage()
		if err != nil {
			h.logger.Info("websocket connection closed",
				"component", "ws.handler",
				"operation", "handle_ws.read_message",
				"user_id", userID,
				"error", err,
			)
			return
		}

		var msg model.ChatMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			h.logger.Warn("invalid websocket message payload",
				"component", "ws.handler",
				"operation", "handle_ws.unmarshal",
				"user_id", userID,
				"error", err,
			)
			continue
		}

		if err := msg.Validate(); err != nil {
			if errors.Is(err, model.ErrChatIDRequired) ||
				errors.Is(err, model.ErrSenderIDRequired) ||
				errors.Is(err, model.ErrBodyRequired) ||
				errors.Is(err, model.ErrBodyTooLong) {
				h.logger.Warn("invalid websocket message payload",
					"component", "ws.handler",
					"operation", "handle_ws.validate",
					"user_id", userID,
					"error", err,
				)
				continue
			}
			h.logger.Error("failed to validate websocket message",
				"component", "ws.handler",
				"operation", "handle_ws.validate",
				"user_id", userID,
				"error", err,
			)
			continue
		}

		if err := h.producer.WriteMessage(context.Background(), []byte(msg.ChatID), payload); err != nil {
			h.logger.Error("failed to publish message to kafka",
				"component", "ws.handler",
				"operation", "handle_ws.publish",
				"user_id", userID,
				"chat_id", msg.ChatID,
				"error", err,
			)
			continue
		}
	}
}
