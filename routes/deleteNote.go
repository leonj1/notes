package routes

import (
	"net/http"
	"notes/clients"
)

func DeleteNote(w http.ResponseWriter, r *http.Request) {
	clients.DeleteNote(w, r)
}
