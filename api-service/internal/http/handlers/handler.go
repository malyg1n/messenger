package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"api-service/internal/model"
	"api-service/internal/service"
)

// Handler объединяет HTTP-обработчики и доступ к бизнес-сервисам.
type Handler struct {
	authSvc    *service.AuthService
	userSvc    *service.UserService
	chatSvc    *service.ChatService
	messageSvc *service.MessageService
	logger     *slog.Logger
}

// New создает HTTP-слой и внедряет зависимости сервисного уровня.
func New(authSvc *service.AuthService, userSvc *service.UserService, chatSvc *service.ChatService, messageSvc *service.MessageService, logger *slog.Logger) *Handler {
	return &Handler{
		authSvc:    authSvc,
		userSvc:    userSvc,
		chatSvc:    chatSvc,
		messageSvc: messageSvc,
		logger:     logger,
	}
}

// Register регистрирует публичные API-маршруты приложения.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/register", h.register)
	mux.HandleFunc("/login", h.login)
	mux.HandleFunc("/users", h.listUsers)
	mux.HandleFunc("/chats/direct", h.createDirectChat)
	mux.HandleFunc("/messages", h.listMessages)
	mux.HandleFunc("/chats", h.listChats)
}

// register создает нового пользователя по username.
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	user, err := h.authSvc.Register(r.Context(), req.Username)
	if err != nil {
		h.logger.Error("register failed", "component", "http.register", "operation", "register", "error", err)
		http.Error(w, "username taken", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, user, h.logger)
}

// login выполняет вход существующего пользователя по username.
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	user, err := h.authSvc.Login(r.Context(), req.Username)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		h.logger.Error("login failed", "component", "http.login", "operation", "login", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, user, h.logger)
}

// listUsers возвращает список зарегистрированных пользователей.
func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	users, err := h.userSvc.List(r.Context())
	if err != nil {
		h.logger.Error("list users failed", "component", "http.users", "operation", "list", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, users, h.logger)
}

// createDirectChat создает личный чат между двумя пользователями или возвращает существующий.
func (h *Handler) createDirectChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req model.CreateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	chatID, err := h.chatSvc.GetOrCreateDirect(r.Context(), req.UserID, req.TargetUserID)
	if err != nil {
		h.logger.Error("direct chat failed", "component", "http.chats.direct", "operation", "create_or_get", "user_id", req.UserID, "target_user_id", req.TargetUserID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, model.ChatResponse{ChatID: chatID}, h.logger)
}

// listMessages возвращает историю сообщений чата с пагинацией по времени.
func (h *Handler) listMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	chatID := r.URL.Query().Get("chat_id")
	limit := r.URL.Query().Get("limit")
	before := r.URL.Query().Get("before")
	if limit == "" {
		limit = "50"
	}

	messages, err := h.messageSvc.ListByChatID(r.Context(), chatID, before, limit)
	if err != nil {
		h.logger.Error("list messages failed", "component", "http.messages", "operation", "list", "chat_id", chatID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, messages, h.logger)
}

// listChats возвращает список чатов пользователя с последними сообщениями.
func (h *Handler) listChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	chats, err := h.chatSvc.ListForUser(r.Context(), userID)
	if err != nil {
		h.logger.Error("list chats failed", "component", "http.chats", "operation", "list", "user_id", userID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, chats, h.logger)
}

// writeJSON сериализует значение и записывает JSON-ответ.
func writeJSON(w http.ResponseWriter, status int, value any, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		logger.Error("encode response failed", "component", "http.response", "operation", "write_json", "status", status, "error", err)
	}
}

// CORS добавляет базовые CORS-заголовки и обрабатывает preflight-запросы.
func CORS(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader перехватывает код ответа для логирования.
func (sw *statusWriter) WriteHeader(statusCode int) {
	sw.status = statusCode
	sw.ResponseWriter.WriteHeader(statusCode)
}

// Logging логирует метод, путь, статус и длительность каждого HTTP-запроса.
func Logging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		logger.Info("http request",
			"component", "http.middleware",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"latency_ms", time.Since(start).Milliseconds(),
		)
	})
}
