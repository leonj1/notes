package models

import (
	"time"
	"fmt"
)

const TagsTable = "tags"

type Tag struct {
	Id 		int64
	NoteId		int64
	Creator		string
	Key 		string
	Value 		string
	CreateDate 	time.Time
}

func (db *DB) AllTags() ([]*Tag, error) {
	sql := fmt.Sprintf("SELECT * from %s", TagsTable)
	rows, err := db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bks := make([]*Tag, 0)

	for rows.Next() {
		tag := new(Tag)
		err := rows.Scan(&tag.Id, &tag.Key, &tag.Value)
		if err != nil {
			return nil, err
		}
		bks = append(bks, tag)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return bks, nil
}

func (tag Tag) Save() (*Tag, error){
	var sql string
	if tag.Id == 0 {
		sql = fmt.Sprintf("INSERT INTO %s (key, value, creator, create_date) VALUES (?,?,?,?)", TagsTable)
	} else {
		sql = fmt.Sprintf("UPDATE %s SET key=?, value=?, creator=?, create_date=? WHERE id=%d", TagsTable, tag.Id)
	}

	res, err := db.Exec(sql, tag.Key, tag.Value, tag.CreateDate)
	if err != nil {
		return nil, err
	}

	if tag.Id == 0 {
		tag.Id, err = res.LastInsertId()
		if err != nil {
			return nil, err
		}
	}

	return &tag, nil
}
