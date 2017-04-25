package routes

import (
	"net/http"
	"notes/models"
	"encoding/json"
)

func DeleteNote(w http.ResponseWriter, r *http.Request) {
	var note models.Note

	if r.Method != "DELETE" {
		http.Error(w, "Only DELETE HTTP method allowed here", http.StatusForbidden)
		return
	}

	err := note.DeleteNodeById()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	msg := []byte(`{"status":"deleted"}`)

	js, err := json.Marshal(&msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, JSON)
	w.Write(js)
}
