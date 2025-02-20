package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/SethGK/chirpy/internal/auth"
	"github.com/SethGK/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	jwtSecret      string
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdateAt  time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type ChirpRequest struct {
	Body string `json:"body"`
}

type CreateChirpRequest struct {
	Body string `json:"body"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	CleanedBody string `json:"cleaned_body"`
}

var profaneWords = []string{"kerfuffle", "sharbert", "fornax"}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not set in .env")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set in .env")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	dbQueries := database.New(db)
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		platform = "prod"
	}

	apiCfg := apiConfig{
		db:        dbQueries,
		platform:  platform,
		jwtSecret: jwtSecret,
	}

	const filepathRoot = "."
	const port = "8080"

	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir(filepathRoot))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", fileServer)))

	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerAdminMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerAdminReset)

	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		handlerCreateChirp(&apiCfg, w, r)
	})

	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		apiCfg.handlerGetAllChirps(w, r)
	})
	mux.HandleFunc("GET /api/chirps/", func(w http.ResponseWriter, r *http.Request) {
		apiCfg.handlerGetChirp(w, r)
	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		apiCfg.handlerLogin(w, r)
	})
	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		apiCfg.handlerRefresh(w, r)
	})
	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		apiCfg.handlerRevoke(w, r)
	})
	mux.HandleFunc("PUT /api/users", func(w http.ResponseWriter, r *http.Request) {
		apiCfg.handlerUpdateUser(w, r)
	})
	mux.HandleFunc("DELETE /api/chirps/", func(w http.ResponseWriter, r *http.Request) {
		apiCfg.handlerDeleteChirp(w, r)
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerAdminMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	hits := cfg.fileserverHits.Load()
	html := fmt.Sprintf(`
	<html>
		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
	</html>`, hits)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

func (cfg *apiConfig) handlerAdminReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	err := cfg.db.DeleteAllUsers(r.Context())
	if err != nil {
		log.Printf("Error deleting users: %s", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, map[string]string{"message": "All users deleted"}, http.StatusOK)
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Invalid request method"}`, http.StatusMethodNotAllowed)
		return
	}

	var req CreateUserRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		log.Printf("Error decoding JSON: %s", err)
		sendJSONResponse(w, ErrorResponse{Error: "Invalid JSON"}, http.StatusBadRequest)
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		sendJSONResponse(w, ErrorResponse{Error: "Failed to process password"}, http.StatusInternalServerError)
		return
	}

	userRes, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email:          req.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		log.Printf("Error creating user: %s", err)
		sendJSONResponse(w, ErrorResponse{Error: "Failed to create user"}, http.StatusInternalServerError)
		return
	}

	user := User{
		ID:        userRes.ID,
		CreatedAt: userRes.CreatedAt,
		UpdateAt:  userRes.UpdatedAt,
		Email:     userRes.Email,
	}

	sendJSONResponse(w, user, http.StatusCreated)
}

// handlerCreateChirp now decodes into CreateChirpRequest so it gets both body and user_id.
func handlerCreateChirp(cfg *apiConfig, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Invalid request method"}`, http.StatusMethodNotAllowed)
		return
	}

	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, err := auth.ValidateJWT(tokenString, cfg.jwtSecret)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateChirpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding JSON: %s", err)
		sendJSONResponse(w, ErrorResponse{Error: "Invalid JSON"}, http.StatusBadRequest)
		return
	}

	if len(req.Body) > 140 {
		sendJSONResponse(w, ErrorResponse{Error: "Chirp is too long"}, http.StatusBadRequest)
		return
	}

	cleanedBody := cleanChirpBody(req.Body)

	dbChirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleanedBody,
		UserID: userID,
	})
	if err != nil {
		log.Printf("Error creating chirp: %s", err)
		sendJSONResponse(w, ErrorResponse{Error: "Failed to create chirp"}, http.StatusInternalServerError)
		return
	}

	apiChirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	sendJSONResponse(w, apiChirp, http.StatusCreated)
}

func cleanChirpBody(body string) string {
	words := strings.Split(body, " ")
	for i, word := range words {
		normalizedWord := strings.ToLower(word)
		for _, profane := range profaneWords {
			if normalizedWord == profane {
				words[i] = "****"
				break
			}
		}
	}
	return strings.Join(words, " ")
}

func sendJSONResponse(w http.ResponseWriter, response interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
}

func respondWithError(w http.ResponseWriter, status int, message string, err error) {
	response := map[string]string{
		"error":   message,
		"details": err.Error(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(response)
}
