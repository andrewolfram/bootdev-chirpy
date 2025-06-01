package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"

	"example.com/username/bootdev-chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

/* SRUCTS */
type parameters struct {
	Body string `json:"body"`
}

type returnVals struct {
	Valid       bool   `json:"valid"`
	CleanedBody string `json:"cleaned_body"`
}

type errorVals struct {
	Error string `json:"error"`
}

type apiConfig struct {
	fileserverHits atomic.Int32
}

/* FUNCTIONS */
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func healthCheck(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) metrics(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	val := cfg.fileserverHits.Load()
	template := `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>
	`
	w.Write([]byte(fmt.Sprintf(template, val)))
}

func (cfg *apiConfig) reset(w http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits = atomic.Int32{}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func sanitize(input string) string {
	bad_words := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Split(input, " ")
	result := []string{}
	for _, word := range words {
		if !slices.Contains(bad_words, strings.ToLower(word)) {
			result = append(result, word)
		} else {
			result = append(result, "****")
		}
	}
	return strings.Join(result, " ")
}

func respondWithError(w http.ResponseWriter, respBody errorVals) {
	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)
	w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, respBody returnVals) {
	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}

func validateChirp(w http.ResponseWriter, req *http.Request) {

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		// an error will be thrown if the JSON is invalid or has the wrong types
		// any missing fields will simply have their values in the struct set to their zero value
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		w.Write([]byte("Error decoding parameters"))
		return
	}

	if len(params.Body) > 140 {
		respBody := errorVals{
			"Chirp is too long",
		}
		respondWithError(w, respBody)
	} else {
		respBody := returnVals{
			true, sanitize(params.Body),
		}
		respondWithJSON(w, respBody)
	}
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, _ := sql.Open("postgres", dbURL)
	dbQueries := database.New(db)
	fmt.Println(dbQueries)

	const port = "8080"

	mux := http.NewServeMux()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	apiCnfg := apiConfig{
		fileserverHits: atomic.Int32{},
	}

	log.Printf("Serving on port: %s\n", port)
	/*Admin stuuf */
	mux.HandleFunc("GET /admin/metrics", apiCnfg.metrics)
	mux.HandleFunc("POST /admin/reset", apiCnfg.reset)

	/* API stuff */
	mux.HandleFunc("GET /api/healthz", healthCheck)
	mux.HandleFunc("POST /api/validate_chirp", validateChirp)

	/* App stuff */
	mux.Handle("/app/", http.StripPrefix("/app", apiCnfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	log.Fatal(srv.ListenAndServe())
}
