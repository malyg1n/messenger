package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

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

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	db, err := sql.Open(
		"postgres",
		"postgres://postgres:postgres@postgres:5432/chat?sslmode=disable",
	)

	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		var req RegisterRequest
		json.NewDecoder(r.Body).Decode(&req)
		id := uuid.New().String()
		_, err := db.Exec("insert into users(id, username) values ($1,$2)", id, req.Username)
		if err != nil {
			http.Error(w, "username taken", 400)
			return

		}

		json.NewEncoder(w).Encode(
			User{
				ID:       id,
				Username: req.Username,
			},
		)

	})

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		rows, err := db.Query(
			"select id, username from users order by created_at desc",
		)

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		defer rows.Close()
		users := make([]User, 0)
		for rows.Next() {
			var u User
			rows.Scan(&u.ID, &u.Username)
			users = append(users, u)
		}

		json.NewEncoder(w).Encode(users)

	})

	mux.HandleFunc("/chats/direct", func(w http.ResponseWriter, r *http.Request) {

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		var req CreateChatRequest

		json.NewDecoder(r.Body).Decode(&req)

		var chatID string

		err := db.QueryRow(`
			select c.id
			from chats c
			join chat_participants p1 on p1.chat_id = c.id
			join chat_participants p2 on p2.chat_id = c.id
	
			where p1.user_id = $1
			and p2.user_id = $2
			limit 1
		`,
			req.UserID,
			req.TargetUserID,
		).Scan(&chatID)

		if err == nil {

			json.NewEncoder(w).Encode(
				ChatResponse{
					ChatID: chatID,
				},
			)

			return
		}

		chatID = uuid.New().String()

		_, err = db.Exec(
			"insert into chats(id) values ($1)",
			chatID,
		)

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		_, err = db.Exec(
			`
			insert into chat_participants
			(chat_id, user_id)
			values
			($1,$2),
			($1,$3)
			`,
			chatID,
			req.UserID,
			req.TargetUserID,
		)

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		json.NewEncoder(w).Encode(
			ChatResponse{
				ChatID: chatID,
			},
		)

	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		var req RegisterRequest

		err := json.NewDecoder(r.Body).Decode(&req)

		if err != nil {
			http.Error(w, "bad json", 400)
			return
		}

		var u User

		err = db.QueryRow(
			`
			select id, username
			from users
			where username = $1
			`,
			req.Username,
		).Scan(&u.ID, &u.Username)

		if err != nil {
			http.Error(w, "user not found", 404)
			return
		}

		json.NewEncoder(w).Encode(u)

	})

	mux.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		chatID := r.URL.Query().Get("chat_id")
		limit := r.URL.Query().Get("limit")

		if limit == "" {
			limit = "50"
		}

		rows, err := db.Query(
			`
			select sender_id, body, created_at
			from messages
			where chat_id = $1
			order by created_at desc
			limit $2
			`,
			chatID,
			limit,
		)

		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		defer rows.Close()

		messages := make([]Message, 0)

		for rows.Next() {

			var m Message

			rows.Scan(
				&m.SenderID,
				&m.Body,
				&m.CreatedAt,
			)

			messages = append(messages, m)

		}

		json.NewEncoder(w).Encode(messages)

	})

	mux.HandleFunc("/chats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "user_id is required", 400)
			return
		}

		rows, err := db.Query(`
			select
				c.id as chat_id,
				u.username as title,
				coalesce(m.body, '') as last_message,
				coalesce(m.created_at::text, '') as last_message_at
			from chats c
			join chat_participants self_p
				on self_p.chat_id = c.id
			join chat_participants other_p
				on other_p.chat_id = c.id
				and other_p.user_id <> self_p.user_id
			join users u
				on u.id = other_p.user_id
			left join lateral (
				select body, created_at
				from messages
				where chat_id = c.id
				order by created_at desc
				limit 1
			) m on true
			where self_p.user_id = $1
			order by m.created_at desc nulls last
		`, userID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		chats := make([]ChatListItem, 0)

		for rows.Next() {
			var item ChatListItem
			err := rows.Scan(
				&item.ChatID,
				&item.Title,
				&item.LastMessage,
				&item.LastMessageAt,
			)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			chats = append(chats, item)
		}

		json.NewEncoder(w).Encode(chats)
	})

	log.Println("api started")
	http.ListenAndServe(":8081", cors(mux))

}
