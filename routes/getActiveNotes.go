package routes

import (
	"net/http"
	"notes/clients"
)

func ActiveNotes(w http.ResponseWriter, r *http.Request) {
	clients.ActiveNotes(w, r)
}
