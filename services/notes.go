package services

import (
	"notes/clients"
	"notes/models"
)

// NotesService provides high-level business operations for notes.
// Routes that need to interact with notes should depend on this service
// rather than calling the clients package directly.
type NotesService struct{}

// NewNotesService returns a new NotesService.
func NewNotesService() *NotesService {
	return &NotesService{}
}

// Add persists a new note (or updates an existing one when Id is non-zero)
// and returns the saved record.
func (s *NotesService) Add(note models.Note) (*models.Note, error) {
	return clients.AddNote(note)
}

// Active returns all notes that are currently active (no expiration, or
// expiration in the future).
func (s *NotesService) Active() ([]*models.Note, error) {
	return clients.ActiveNotes()
}

// All returns every note row.
func (s *NotesService) All() ([]*models.Note, error) {
	return clients.AllNotes()
}

// Delete removes the note (and its tags) identified by id.
func (s *NotesService) Delete(id int64) error {
	return clients.DeleteNote(id)
}

// FilterByTag returns all notes that have a tag matching the given
// key/value pair.
func (s *NotesService) FilterByTag(key, value string) ([]*models.Note, error) {
	return clients.FilterNotesByTag(key, value)
}

// ErrTagAlreadyExists is re-exported from the clients package so callers
// of the service do not need to import clients to inspect this sentinel.
var ErrTagAlreadyExists = clients.ErrTagAlreadyExists

// AddTag attaches a new tag to a note. The supplied tag must have Key,
// Value, and NoteId populated. If a tag with the same Key/Value already
// exists on the note, ErrTagAlreadyExists is returned.
func (s *NotesService) AddTag(tag models.Tag) (*models.Tag, error) {
	return clients.AddTag(tag)
}

// Notes is the default package-level NotesService instance used by route
// handlers.
var Notes = NewNotesService()
