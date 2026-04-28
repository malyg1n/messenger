package dto

import "api-service/internal/model"

type AuthResponse struct {
	User  model.User   `json:"user"`
	Token string `json:"token"`
}