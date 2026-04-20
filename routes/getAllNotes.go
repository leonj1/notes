package routes

import (
	"encoding/json"
	"net/http"
	"notes/clients"
)

func AllNotes(w http.ResponseWriter, r *http.Request) {
	notes, err := clients.AllNotes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(notes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, JSON)
	w.Write(js)
}
