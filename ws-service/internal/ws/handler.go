package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"ws-service/auth"
	"ws-service/internal/broker"
	"ws-service/internal/model"
)

const kafkaPublishTimeout = 10 * time.Second

// Handler обслуживает websocket-подключения и отправляет сообщения в Kafka.
type Handler struct {
	producer broker.Producer
	hub      *Hub
	logger   *slog.Logger
	upgrader websocket.Upgrader
	jwtSvc   *auth.JWTService
}

// NewHandler создает websocket-handler с доступом к Kafka producer и Hub.
func NewHandler(producer broker.Producer, hub *Hub, logger *slog.Logger, jwtSvc *auth.JWTService) *Handler {
	return &Handler{
		producer: producer,
		hub:      hub,
		logger:   logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
		jwtSvc: jwtSvc,
	}
}

// HandleWS обрабатывает lifecycle websocket-сессии конкретного пользователя.
func (h *Handler) HandleWS(w http.ResponseWriter, r *http.Request) {
	// user_id — ключ маршрутизации в Hub: для пользователя держим одно активное подключение.
	// userID := r.URL.Query().Get("user_id")
	token := r.URL.Query().Get("token")
	parsedToken, err := h.jwtSvc.Parse(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		h.logger.Warn("invalid token",
			"component", "ws.handler",
			"operation", "handle_ws.validate_token",
			"error", err,
		)
		return
	}

	userID := parsedToken.Subject
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
		msg.SenderID = userID

		if err := msg.Validate(); err != nil {
			h.logger.Error("failed to validate websocket message",
				"component", "ws.handler",
				"operation", "handle_ws.validate",
				"user_id", userID,
				"error", err,
			)
			continue
		}

		pubCtx, cancel := context.WithTimeout(r.Context(), kafkaPublishTimeout)
		werr := h.producer.WriteMessage(pubCtx, []byte(msg.ChatID), msg)
		cancel()
		if werr != nil {
			h.logger.Error("failed to publish message to kafka",
				"component", "ws.handler",
				"operation", "handle_ws.publish",
				"user_id", userID,
				"chat_id", msg.ChatID,
				"error", werr,
			)
			continue
		}
	}
}
