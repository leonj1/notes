package routes

import (
	"encoding/json"
	"errors"
	"github.com/husobee/vestigo"
	"net/http"
	"notes/models"
	"notes/services"
	"strconv"
)

func AddTags(w http.ResponseWriter, r *http.Request) {
	id := vestigo.Param(r, "id")
	if id == "" {
		http.Error(w, "Invalid note_id", http.StatusBadRequest)
		return
	}

	noteId, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		http.Error(w, "Invalid note_id", http.StatusBadRequest)
		return
	}

	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}

	var tag models.Tag
	if err := json.NewDecoder(r.Body).Decode(&tag); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tag.NoteId = noteId

	saved, err := services.Notes.AddTag(tag)
	if err != nil {
		if errors.Is(err, services.ErrTagAlreadyExists) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(&saved)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, JSON)
	w.Write(js)
}
