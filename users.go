package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"example.com/username/bootdev-chirpy/internal/auth"
	"example.com/username/bootdev-chirpy/internal/database"
)

func (cfg *apiConfig) updateUser(w http.ResponseWriter, req *http.Request) {
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
	hash, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Error creating hash %v", err)))
		fmt.Println(err)
		return
	}
	param_struct := database.UpdateUserParams{ID: userId, Password: hash, Email: params.Email}
	user, err := cfg.db.UpdateUser(req.Context(), param_struct)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Error creating user %v", err)))
		fmt.Println(err)
		return
	}
	user_struct := User{user.ID, user.CreatedAt, user.UpdatedAt, user.Email, "", "", user.IsChirpyRed.Bool}
	respondWithJSON(w, 200, user_struct)
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error decoding parameters"))
		return
	}

	if params.Email != "" || params.Password != "" {
		hash, err := auth.HashPassword(params.Password)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("Error creating hash %v", err)))
			fmt.Println(err)
			return
		}
		param_struct := database.CreateUserParams{Password: hash, Email: params.Email}
		user, err := cfg.db.CreateUser(req.Context(), param_struct)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("Error creating user %v", err)))
			fmt.Println(err)
			return
		}
		user_struct := User{user.ID, user.CreatedAt, user.UpdatedAt, user.Email, "", "", user.IsChirpyRed.Bool}
		respondWithJSON(w, 201, user_struct)
	}

}
