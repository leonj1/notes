package routes

import (
	"encoding/json"
	"github.com/husobee/vestigo"
	"net/http"
	"notes/services"
)

func FilterNotesByTag(w http.ResponseWriter, r *http.Request) {
	key := vestigo.Param(r, "key")
	if key == "" {
		http.Error(w, "Invalid key", http.StatusBadRequest)
		return
	}
	value := vestigo.Param(r, "value")
	if value == "" {
		http.Error(w, "Invalid value", http.StatusBadRequest)
		return
	}

	notes, err := services.Notes.FilterByTag(key, value)
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
