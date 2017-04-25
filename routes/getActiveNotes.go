package routes

import (
	"encoding/json"
	"net/http"
	"notes/models"
)

func ActiveNotes(w http.ResponseWriter, r *http.Request) {
	var n models.Note
	notes, err := n.GetActiveNotes()
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
