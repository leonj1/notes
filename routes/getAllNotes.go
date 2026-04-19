package routes

import (
	"net/http"
	"notes/clients"
)

func AllNotes(w http.ResponseWriter, r *http.Request) {
	clients.AllNotes(w, r)
}
