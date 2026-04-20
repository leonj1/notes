package clients

import (
	"errors"
	"notes/models"
)

// ErrTagAlreadyExists is returned by AddTag when a tag with the same
// key/value already exists for the supplied note id.
var ErrTagAlreadyExists = errors.New("tag already exists for this node")

// AddTag persists a new tag for a note after verifying that an identical
// tag (same Key, Value, and NoteId) does not already exist. The supplied
// tag must have Key, Value, and NoteId populated.
func AddTag(tag models.Tag) (*models.Tag, error) {
	existing, err := tag.FindByKeyAndValueAndNoteId(tag.Key, tag.Value, tag.NoteId)
	if err != nil {
		return nil, err
	}

	if existing != nil && len(*existing) > 0 {
		return nil, ErrTagAlreadyExists
	}

	return tag.Save()
}
