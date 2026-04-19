package routes

import (
	"net/http"
	"notes/clients"
)

func FilterNotesByTag(w http.ResponseWriter, r *http.Request) {
	clients.FilterNotesByTag(w, r)
}
