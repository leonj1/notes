package models

import (
	"time"
	"fmt"
	"log"
)

const NotesTable = "notes"

type Note struct {
	Id 		int64
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

func (note Note) Save() (int64, error){
	var sql string
	if note.Id == 0 {
		note.CreateDate = time.Now()
		sql = fmt.Sprintf("INSERT INTO %s (note, create_date, expiration_date) VALUES ('%s', '%s', '%s')", NotesTable, note.Note, note.CreateDate, note.ExpirationDate)
	} else {
		sql = fmt.Sprintf("UPDATE %s SET note='%s', create_date='%s', expiration_date='%s' WHERE id=%d", NotesTable, note.Note, note.CreateDate, note.ExpirationDate, note.Id)
	}

	// TODO Delete me since its for debugging
	log.Println(sql)

	res, err := db.Exec(sql)
	if err != nil {
		return 0, err
	}

	if note.Id == 0 {
		note.Id, err = res.LastInsertId()
		if err != nil {
			return 0, err
		}
	}

	return note.Id, nil
}
