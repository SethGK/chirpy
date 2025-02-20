package main

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/SethGK/chirpy/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	chirpIDStr := strings.TrimPrefix(r.URL.Path, "/api/chirps/")
	if chirpIDStr == "" {
		http.Error(w, "ChirpID not provided", http.StatusBadRequest)
		return
	}

	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		http.Error(w, "Invalid chirp ID", http.StatusBadRequest)
		return
	}

	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := auth.ValidateJWT(accessToken, cfg.jwtSecret)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Chirp not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if chirp.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	err = cfg.db.DeleteChirp(r.Context(), chirpID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
