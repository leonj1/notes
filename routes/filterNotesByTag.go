package routes

import (
	"encoding/json"
	"net/http"
	"notes/models"
	"github.com/husobee/vestigo"
)

func FilterNotesByTag(w http.ResponseWriter, r *http.Request) {
	var t models.Tag
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

	tags, err := t.FindByKeyAndValue(key, value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ids := new([]Tag)

	js, err := json.Marshal(notes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, JSON)
	w.Write(js)
}
