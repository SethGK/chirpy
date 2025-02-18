package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/SethGK/chirpy/internal/auth"
	"github.com/google/uuid"
)

type LoginRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	ExpiresInSeconds int64  `json:"expires_in_seconds,omitempty"`
}

type LoginResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Invalid request method"}`, http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding login JSON: %s", err)
		sendJSONResponse(w, ErrorResponse{Error: "Invalid JSON"}, http.StatusBadRequest)
		return
	}

	user, err := cfg.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Incorrect email or password", http.StatusUnauthorized)
			return
		}
		log.Printf("Error retrieving user: %s", err)
		sendJSONResponse(w, ErrorResponse{Error: "Failed to login"}, http.StatusInternalServerError)
		return
	}

	if err := auth.CheckPasswordHash(req.Password, user.HashedPassword); err != nil {
		http.Error(w, "Incorrect email or password", http.StatusUnauthorized)
		return
	}

	expiresInSec := req.ExpiresInSeconds
	if expiresInSec <= 0 || expiresInSec > 3600 {
		expiresInSec = 3600
	}
	expiresIn := time.Duration(expiresInSec) * time.Second

	token, err := auth.MakeJWT(user.ID, cfg.jwtSecret, expiresIn)
	if err != nil {
		log.Printf("Error creating JWT: %s", err)
		sendJSONResponse(w, ErrorResponse{Error: "Failed to create token"}, http.StatusInternalServerError)
		return
	}

	res := LoginResponse{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		Token:     token,
	}
	sendJSONResponse(w, res, http.StatusOK)

}
