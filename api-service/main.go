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
		var users []User
		for rows.Next() {
			var u User
			rows.Scan(&u.ID, &u.Username)
			users = append(users, u)
		}

		json.NewEncoder(w).Encode(users)

	})

	log.Println("api started")
	http.ListenAndServe(":8081", cors(mux))

}
