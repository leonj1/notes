package models

import (
	"time"
	"fmt"
)

const NotesTable = "notes"

type Note struct {
	Id 		int
	Note 		string
	CreateDate 	time.Time
	ExpirationDate 	time.Time
}

func (db *DB) AllNotes() ([]*Note, error) {
	sql := fmt.Sprintf("SELECT * from %s", NotesTable)
	rows, err := db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bks := make([]*Note, 0)

	for rows.Next() {
		note := new(Note)
		err := rows.Scan(&note.Id, &note.Note)
		if err != nil {
			return nil, err
		}
		bks = append(bks, note)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return bks, nil
}

func (n Note) Save(note Note) (error){
	var sql string
	if note.Id == 0 {
		sql = fmt.Sprintf("INSERT INTO %s (note, create_date, expiration_date) VALUES ('%s', %t, %t)", NotesTable, note.Id, note.Note, note.CreateDate, note.ExpirationDate)
	} else {
		sql = fmt.Sprintf("UPDATE %s SET note='%s', create_date=%t, expiration_date=%t WHERE id=%d", NotesTable, note.Note, note.CreateDate, note.ExpirationDate, note.Id)
	}
	rows, err := db.Query(sql)
	if err != nil {
		return err
	}

	defer rows.Close()

	return nil
}
