package main

import (
	"database/sql"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/SethGK/chirpy/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Invalid request method"}`, http.StatusMethodNotAllowed)
		return
	}

	authorIDstr := r.URL.Query().Get("author_id")

	var (
		dbChirps []database.Chirp
		err      error
	)

	if authorIDstr != "" {
		authorID, parseErr := uuid.Parse(authorIDstr)
		if parseErr != nil {
			http.Error(w, "Invalid author_id", http.StatusBadRequest)
			return
		}
		dbChirps, err = cfg.db.GetChirpsByAuthor(r.Context(), authorID)
	} else {
		dbChirps, err = cfg.db.GetAllChirps(r.Context())
	}

	if err != nil {
		log.Printf("Error retrieving chirps: %s", err)
		sendJSONResponse(w, ErrorResponse{Error: "Failed to retrieve chirps"}, http.StatusInternalServerError)
		return
	}

	var chirps []Chirp
	for _, c := range dbChirps {
		chirps = append(chirps, Chirp{
			ID:        c.ID,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
			Body:      c.Body,
			UserID:    c.UserID,
		})
	}

	sortParam := r.URL.Query().Get("sort")
	if strings.ToLower(sortParam) == "desc" {
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[j].CreatedAt.Before(chirps[i].CreatedAt)
		})
	} else {
		sort.Slice(chirps, func(i, j int) bool {
			return chirps[i].CreatedAt.Before(chirps[j].CreatedAt)
		})
	}

	sendJSONResponse(w, chirps, http.StatusOK)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Invalid request method"}`, http.StatusMethodNotAllowed)
		return
	}

	chirpIDStr := strings.TrimPrefix(r.URL.Path, "/api/chirps/")
	if chirpIDStr == "" || chirpIDStr == r.URL.Path {
		http.NotFound(w, r)
		return
	}

	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid chirp ID"}`, http.StatusBadRequest)
		return
	}

	dbChirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		log.Printf("Error retrieving chirp: %s", err)
		sendJSONResponse(w, ErrorResponse{Error: "Failed to retrieve chirp"}, http.StatusInternalServerError)
		return
	}

	apiChirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	sendJSONResponse(w, apiChirp, http.StatusOK)
}
