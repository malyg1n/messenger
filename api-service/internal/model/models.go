package model

type RegisterRequest struct {
	Username string `json:"username"`
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type CreateChatRequest struct {
	UserID       string `json:"user_id"`
	TargetUserID string `json:"target_user_id"`
}

type ChatResponse struct {
	ChatID string `json:"chat_id"`
}

type Message struct {
	SenderID  string `json:"sender_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

type ChatListItem struct {
	ChatID        string `json:"chat_id"`
	Title         string `json:"title"`
	LastMessage   string `json:"last_message"`
	LastMessageAt string `json:"last_message_at"`
}
