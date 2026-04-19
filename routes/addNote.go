package routes

import (
	"net/http"
	"notes/clients"
)

const ContentType = "Content-Type"
const JSON = "application/json"

func AddNote(w http.ResponseWriter, r *http.Request) {
	clients.AddNote(w, r)
}
