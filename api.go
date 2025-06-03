package main

/*Stuff related to API routes*/

import (
	"encoding/json"
	"net/http"
	"time"

	"example.com/username/bootdev-chirpy/internal/auth"
	"example.com/username/bootdev-chirpy/internal/database"
)

const (
	defaultExpiresIn int = 3600
)

func (cfg *apiConfig) login(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error decoding parameters"))
		return
	}

	user, err := cfg.db.GetUserByMail(req.Context(), params.Email)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Incorrect email or password"))
		return
	}
	err = auth.CheckPasswordHash(user.Password, params.Password)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Incorrect email or password"))
		return
	}
	expiresIn := defaultExpiresIn
	token, err := auth.MakeJWT(user.ID, cfg.secret, time.Duration(time.Duration(expiresIn)*time.Second))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to generate token..."))
		return
	}
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to generate token..."))
		return
	}
	_, err = cfg.db.CreateRefreshToken(req.Context(), database.CreateRefreshTokenParams{
		UserID: user.ID,
		Token:  refreshToken,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save refresh token", err)
		return
	}

	user_struct := User{user.ID, user.CreatedAt, user.UpdatedAt, user.Email, token, refreshToken, user.IsChirpyRed.Bool}
	respondWithJSON(w, http.StatusOK, user_struct)
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token string `json:"token"`
	}

	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find token", err)
		return
	}

	user, err := cfg.db.GetUserByRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't get user for refresh token", err)
		return
	}

	accessToken, err := auth.MakeJWT(
		user.ID,
		cfg.secret,
		time.Hour,
	)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Token: accessToken,
	})
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find token", err)
		return
	}

	_, err = cfg.db.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't revoke session", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
