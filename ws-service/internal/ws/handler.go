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
const presenceRefreshInterval = 10 * time.Second

type presenceTracker interface {
	SetOnline(ctx context.Context, userID string) error
	RefreshOnline(ctx context.Context, userID string) error
	SetOffline(ctx context.Context, userID string) error
}

// Handler обслуживает websocket-подключения и отправляет сообщения в Kafka.
type Handler struct {
	producer broker.Producer
	presence presenceTracker
	hub      *Hub
	logger   *slog.Logger
	upgrader websocket.Upgrader
	jwtSvc   *auth.JWTService
}

// NewHandler создает websocket-handler с доступом к Kafka producer и Hub.
func NewHandler(producer broker.Producer, hub *Hub, logger *slog.Logger, jwtSvc *auth.JWTService, presence presenceTracker) *Handler {
	return &Handler{
		producer: producer,
		presence: presence,
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
	if h.presence != nil {
		if err := h.presence.SetOnline(r.Context(), userID); err != nil {
			h.logger.Warn("failed to set user presence online",
				"component", "ws.handler",
				"operation", "handle_ws.presence_online",
				"user_id", userID,
				"error", err,
			)
		}
		defer func() {
			if err := h.presence.SetOffline(context.Background(), userID); err != nil {
				h.logger.Warn("failed to set user presence offline",
					"component", "ws.handler",
					"operation", "handle_ws.presence_offline",
					"user_id", userID,
					"error", err,
				)
			}
		}()

		stopRefresh := make(chan struct{})
		defer close(stopRefresh)
		go h.refreshPresenceLoop(stopRefresh, userID)
	}

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

func (h *Handler) refreshPresenceLoop(stop <-chan struct{}, userID string) {
	ticker := time.NewTicker(presenceRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if err := h.presence.RefreshOnline(context.Background(), userID); err != nil {
				h.logger.Warn("failed to refresh user presence",
					"component", "ws.handler",
					"operation", "handle_ws.presence_refresh",
					"user_id", userID,
					"error", err,
				)
			}
		}
	}
}
