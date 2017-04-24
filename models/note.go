package models

import (
	"time"
	"fmt"
)

const NotesTable = "notes"

type Note struct {
	Id 		int64
	Note 		string
	Creator		string
	CreateDate 	time.Time
	ExpirationDate 	time.Time
}

func (note Note) AllNotes() ([]*Note, error) {
	sql := fmt.Sprintf("SELECT * from %s", NotesTable)
	rows, err := db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := make([]*Note, 0)

	for rows.Next() {
		note := new(Note)
		err := rows.Scan(&note.Id, &note.Note)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return notes, nil
}

func (note Note) Save() (*Note, error){
	var sql string
	if note.Id == 0 {
		note.CreateDate = time.Now()
		note.CreateDate.Format(time.RFC3339)
		sql = fmt.Sprintf("INSERT INTO %s (note, creator, create_date, expiration_date) VALUES (?,?,?,?)", NotesTable)
	} else {
		sql = fmt.Sprintf("UPDATE %s SET note=?, creator=?, create_date=?, expiration_date=? WHERE id=%d", NotesTable, note.Id)
	}

	res, err := db.Exec(sql, note.Note, note.Creator, note.CreateDate, note.ExpirationDate)
	if err != nil {
		return nil, err
	}

	if note.Id == 0 {
		note.Id, err = res.LastInsertId()
		if err != nil {
			return nil, err
		}
	}

	return &note, nil
}

func (note Note) GetActiveNotes() ([]*Note, error) {
	currentTime := time.Now()
	currentTime.Format(time.RFC3339)
	sql := fmt.Sprintf("SELECT `id`, `note`, `creator`, `create_date`, `expiration_date` from %s where `expiration_date` = '0000-00-00 00:00:00' or expiration_date > ?", NotesTable)
	rows, err := db.Query(sql, currentTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := make([]*Note, 0)

	for rows.Next() {
		note := new(Note)
		err := rows.Scan(&note.Id, &note.Note, &note.Creator, &note.CreateDate, &note.ExpirationDate)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return notes, nil
}
