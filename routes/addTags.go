package routes

import (
	"net/http"
	"notes/clients"
)

func AddTags(w http.ResponseWriter, r *http.Request) {
	clients.AddTags(w, r)
}
