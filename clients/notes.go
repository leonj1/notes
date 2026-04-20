package clients

import (
	"notes/models"
)

// AddNote persists a new note (or updates an existing one when Id is non-zero).
func AddNote(note models.Note) (*models.Note, error) {
	return note.Save()
}

// ActiveNotes returns all notes that are currently active
// (no expiration, or expiration in the future).
func ActiveNotes() ([]*models.Note, error) {
	var n models.Note
	return n.ActiveNotes()
}

// AllNotes returns every note row.
func AllNotes() ([]*models.Note, error) {
	var n models.Note
	return n.AllNotes()
}

// DeleteNote removes the note (and its tags) for the given note id.
func DeleteNote(id int64) error {
	var n models.Note
	return n.DeleteNodeById(id)
}

// FilterNotesByTag returns all notes that have a tag matching the given
// key and value pair.
func FilterNotesByTag(key, value string) ([]*models.Note, error) {
	var t models.Tag
	var n models.Note

	tags, err := t.FindByKeyAndValue(key, value)
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(tags))
	for _, tag := range tags {
		ids = append(ids, tag.NoteId)
	}

	return n.FindIn(&ids)
}
