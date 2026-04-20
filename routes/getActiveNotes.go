package routes

import (
	"encoding/json"
	"net/http"
	"notes/clients"
)

func ActiveNotes(w http.ResponseWriter, r *http.Request) {
	notes, err := clients.ActiveNotes()
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
