package main

import (
	"encoding/json"
	"net/http"

	"example.com/username/bootdev-chirpy/internal/auth"
	"github.com/google/uuid"
)

const upgradeEvent string = "user.upgraded"

type WebhookParams struct {
	Event string `json:"event"`
	Data  struct {
		UserID uuid.UUID `json:"user_id"`
	} `json:"data"`
}

func (cfg *apiConfig) upgradeUser(w http.ResponseWriter, req *http.Request) {
	header, err := auth.GetAPIKey(req.Header)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Error fetching API Key"))
		return
	}
	if header != cfg.polkaKey {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("wrong API Key"))
		return
	}

	decoder := json.NewDecoder(req.Body)
	params := WebhookParams{}
	err = decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error decoding parameters"))
		return
	}
	if params.Event != upgradeEvent {
		w.WriteHeader(http.StatusNoContent)
		return
	} else {
		_, err := cfg.db.UpgradeUser(req.Context(), params.Data.UserID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
