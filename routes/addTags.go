package routes

import (
	"net/http"
	"notes/models"
	"encoding/json"
)

func AddTags(w http.ResponseWriter, r *http.Request) {
	var tag models.Tag
	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tags, err := tag.FindByKeyAndValueAndNoteId(tag.Key, tag.Value, tag.NoteId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(*tags) > 0 {
		http.Error(w, "Tag already exists for this node", http.StatusBadRequest)
		return
	}

	saved, err := tag.Save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
