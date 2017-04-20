package models

import (
	"time"
	"fmt"
)

const TagsTable = "tags"

type Tag struct {
	Id 		int
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

func (t Tag) Save(tag Tag) (error){
	var sql string
	if tag.Id == 0 {
		sql = fmt.Sprintf("INSERT INTO %s (key, value, create_date) VALUES ('%s', %t, %t)", TagsTable, tag.Key, tag.Value, tag.CreateDate)
	} else {
		sql = fmt.Sprintf("UPDATE %s SET key='%s', value='%s', create_date=%t WHERE id=%d", TagsTable, tag.Key, tag.Value, tag.CreateDate, tag.Id)
	}

	rows, err := db.Query(sql)

	if err != nil {
		return err
	}

	defer rows.Close()

	return nil
}
