package handlers

import (
	"2026champs/internal/app"
	"encoding/json"
	"net/http"
)

func GetPlayers(app *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Example: Fetch all players (stub, replace with actual query)
		players := []*struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			{ID: "1", Name: "Player1"},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(players)
	}
}
