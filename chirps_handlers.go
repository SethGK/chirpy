package main

import (
	"log"
	"net/http"
)

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Invalid request method"}`, http.StatusMethodNotAllowed)
		return
	}

	chirps, err := cfg.db.GetAllChirps(r.Context())
	if err != nil {
		log.Printf("Error retrieving chirps: %s", err)
		sendJSONResponse(w, ErrorResponse{Error: "Failed to retrieve chirps"}, http.StatusInternalServerError)
		return
	}

	var apiChirps []Chirp
	for _, c := range chirps {
		apiChirps = append(apiChirps, Chirp{
			ID:        c.ID,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
			Body:      c.Body,
			UserID:    c.UserID,
		})
	}

	sendJSONResponse(w, apiChirps, http.StatusOK)
}
