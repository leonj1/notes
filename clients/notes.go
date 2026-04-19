package clients

import (
	"encoding/json"
	"github.com/husobee/vestigo"
	"net/http"
	"notes/models"
	"strconv"
)

const ContentType = "Content-Type"
const JSON = "application/json"

func AddNote(w http.ResponseWriter, r *http.Request) {
	var note models.Note
	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&note)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	saved, err := note.Save()
	if err != nil {
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

func FilterNotesByTag(w http.ResponseWriter, r *http.Request) {
	var t models.Tag
	var n models.Note

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

	ids := make([]int64, 0)
	for _, tag := range tags {
		ids = append(ids, tag.NoteId)
	}

	notes, err := n.FindIn(&ids)
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

func ActiveNotes(w http.ResponseWriter, r *http.Request) {
	var n models.Note
	notes, err := n.ActiveNotes()
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

func AllNotes(w http.ResponseWriter, r *http.Request) {
	var n models.Note
	notes, err := n.AllNotes()
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

func DeleteNote(w http.ResponseWriter, r *http.Request) {
	var note models.Note

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

	err = note.DeleteNodeById(noteId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO Verify the returned json looks good to the client
	msg := []byte(`{"status":"deleted"}`)

	js, err := json.Marshal(&msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, JSON)
	w.Write(js)
}
