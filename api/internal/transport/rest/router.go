package rest

import (
	"2026champs/internal/app"
	"2026champs/internal/transport/rest/handlers"
	"net/http"

	"github.com/gorilla/mux"
)

func NewRouter(app *app.App) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/players", handlers.GetPlayers(app)).Methods("GET")

	return r
}
