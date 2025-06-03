package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"

	"example.com/username/bootdev-chirpy/internal/auth"
	"example.com/username/bootdev-chirpy/internal/database"
	"github.com/google/uuid"
)

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

func healthCheck(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) getChirp(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("chirpID")
	fmt.Println(id)
	uid, _ := uuid.Parse(id)
	chirp, err := cfg.db.GetChirp(req.Context(), uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No chirp found with that ID
			msg := "Chirp not found"

			respondWithError(w, 404, msg, err)
		}
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Error getting chirp %v", err)))
		fmt.Println(err)
		return
	}
	chirp_struct := Chirp{chirp.ID, chirp.CreatedAt, chirp.UpdatedAt, chirp.Body, chirp.UserID.UUID}
	respondWithJSON(w, 200, chirp_struct)
}

func (cfg *apiConfig) getChirps(w http.ResponseWriter, req *http.Request) {
	a_id := req.URL.Query().Get("author_id")
	sort_order := req.URL.Query().Get("sort")
	if sort_order == "" {
		sort_order = "asc"
	}
	chirps, err := cfg.db.GetAllChirps(req.Context())
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Error creating chirp %v", err)))
		fmt.Println(err)
		return
	}
	chirp_structs := []Chirp{}
	for _, chirp := range chirps {
		chirp_struct := Chirp{chirp.ID, chirp.CreatedAt, chirp.UpdatedAt, chirp.Body, chirp.UserID.UUID}
		chirp_structs = append(chirp_structs, chirp_struct)
	}
	if a_id != "" {
		author_uuid, _ := uuid.Parse(a_id)
		chirp_structs = filterChirpsByUserID(chirp_structs, author_uuid)
	}
	if sort_order == "desc" {
		sort.Slice(chirp_structs, func(i, j int) bool {
			return chirp_structs[i].CreatedAt.After(chirp_structs[j].CreatedAt)
		})
	} else {
		sort.Slice(chirp_structs, func(i, j int) bool {
			return chirp_structs[i].CreatedAt.Before(chirp_structs[j].CreatedAt)
		})
	}
	respondWithJSON(w, 200, chirp_structs)
}

func filterChirpsByUserID(chirps []Chirp, userID uuid.UUID) []Chirp {
	var filtered []Chirp
	for _, chirp := range chirps {
		if chirp.UserID == userID {
			filtered = append(filtered, chirp)
		}
	}
	return filtered
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, req *http.Request) {

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error decoding parameters"))
		return
	}
	token, err := auth.GetBearerToken((req.Header))
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Invalid Token Header!"))
		return
	}
	userId, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Invalid Token Header!"))
		return
	}
	fmt.Println(userId)

	if len(params.Body) > 140 {
		err_msg := "Chirp is too long"
		respondWithError(w, 500, err_msg, errors.New(err_msg))
	} else {
		chirpParams := database.CreateChirpParams{Body: sanitize(params.Body), UserID: uuid.NullUUID{UUID: userId, Valid: true}}
		chirp, err := cfg.db.CreateChirp(req.Context(), chirpParams)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("Error creating chirp %v", err)))
			fmt.Println(err)
			return
		}
		chirp_struct := Chirp{chirp.ID, chirp.CreatedAt, chirp.UpdatedAt, chirp.Body, chirp.UserID.UUID}
		respondWithJSON(w, 201, chirp_struct)
	}
}

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("chirpID")
	fmt.Println(id)
	uid, _ := uuid.Parse(id)
	chirp, err := cfg.db.GetChirp(req.Context(), uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No chirp found with that ID
			msg := "Chirp not found"

			respondWithError(w, 404, msg, err)
		}
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Error getting chirp %v", err)))
		fmt.Println(err)
		return
	}
	fmt.Println(chirp)

	token, err := auth.GetBearerToken((req.Header))
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Invalid Token Header!"))
		return
	}
	userId, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Invalid Token Header!"))
		return
	}

	if (chirp.UserID != uuid.NullUUID{UUID: userId, Valid: true}) {
		msg := "you do not own this Chirp"
		respondWithError(w, 403, msg, errors.New(msg))
	}
	chirpParams := database.DeleteChirpByIDAndUserIDParams{ID: uid, UserID: uuid.NullUUID{UUID: userId, Valid: true}}
	err = cfg.db.DeleteChirpByIDAndUserID(req.Context(), chirpParams)
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf(nil, "Error getting chirp %v", err))
		fmt.Println(err)
		return
	}
	type msgResponse struct {
		Msg string `json:"msg"`
	}
	respondWithJSON(w, 204, msgResponse{
		Msg: "Chirp Deleted!",
	})
}
