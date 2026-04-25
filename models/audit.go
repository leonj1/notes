package models

import (
	"fmt"
	"github.com/kataras/go-errors"
	"time"
)

const ScheduleAuditTable = "schedule_audit"

const (
	AuditStatusSuccess = "success"
	AuditStatusFailure = "failure"
)

type Audit struct {
	Id         int64     `json:"id,string,omitempty"`
	ScheduleId int64     `json:"schedule_id,string,omitempty"`
	ScriptPath string    `json:"script_path,omitempty"`
	Status     string    `json:"status,omitempty"`
	Output     string    `json:"output,omitempty"`
	Error      string    `json:"error,omitempty"`
	StartTime  time.Time `json:"start_time,omitempty"`
	EndTime    time.Time `json:"end_time,omitempty"`
}

func (a Audit) Save() (*Audit, error) {
	if a.ScheduleId == 0 {
		return nil, errors.New("schedule_id is required")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (`schedule_id`, `script_path`, `status`, `output`, `error`, `start_time`, `end_time`) VALUES (?,?,?,?,?,?,?)",
		ScheduleAuditTable,
	)
	res, err := db.Exec(query,
		a.ScheduleId,
		a.ScriptPath,
		a.Status,
		a.Output,
		a.Error,
		a.StartTime,
		a.EndTime,
	)
	if err != nil {
		return nil, err
	}

	a.Id, err = res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (a Audit) FindById(id int64) (*Audit, error) {
	if id < 1 {
		return nil, errors.New("Please provide a valid id")
	}

	query := fmt.Sprintf(
		"SELECT `id`, `schedule_id`, `script_path`, `status`, `output`, `error`, `start_time`, `end_time` FROM %s WHERE `id`=?",
		ScheduleAuditTable,
	)
	row := db.QueryRow(query, id)

	audit := new(Audit)
	err := row.Scan(
		&audit.Id,
		&audit.ScheduleId,
		&audit.ScriptPath,
		&audit.Status,
		&audit.Output,
		&audit.Error,
		&audit.StartTime,
		&audit.EndTime,
	)
	if err != nil {
		return nil, err
	}
	return audit, nil
}

func (a Audit) FindByScheduleId(scheduleId int64) ([]*Audit, error) {
	if scheduleId < 1 {
		return nil, errors.New("Please provide a valid schedule_id")
	}

	query := fmt.Sprintf(
		"SELECT `id`, `schedule_id`, `script_path`, `status`, `output`, `error`, `start_time`, `end_time` FROM %s WHERE `schedule_id`=? ORDER BY `start_time` DESC",
		ScheduleAuditTable,
	)
	rows, err := db.Query(query, scheduleId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAudits(rows)
}

func (a Audit) FindByDateRange(start, end time.Time) ([]*Audit, error) {
	query := fmt.Sprintf(
		"SELECT `id`, `schedule_id`, `script_path`, `status`, `output`, `error`, `start_time`, `end_time` FROM %s WHERE `start_time` >= ? AND `start_time` <= ? ORDER BY `start_time` DESC",
		ScheduleAuditTable,
	)
	rows, err := db.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAudits(rows)
}

func (a Audit) FindRecent(limit int) ([]*Audit, error) {
	if limit < 1 {
		return nil, errors.New("limit must be at least 1")
	}

	query := fmt.Sprintf(
		"SELECT `id`, `schedule_id`, `script_path`, `status`, `output`, `error`, `start_time`, `end_time` FROM %s ORDER BY `start_time` DESC LIMIT ?",
		ScheduleAuditTable,
	)
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAudits(rows)
}

func scanAudits(rows rowScanner) ([]*Audit, error) {
	audits := make([]*Audit, 0)
	for rows.Next() {
		audit := new(Audit)
		err := rows.Scan(
			&audit.Id,
			&audit.ScheduleId,
			&audit.ScriptPath,
			&audit.Status,
			&audit.Output,
			&audit.Error,
			&audit.StartTime,
			&audit.EndTime,
		)
		if err != nil {
			return nil, err
		}
		audits = append(audits, audit)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return audits, nil
}

// rowScanner abstracts *sql.Rows so scanAudits can be reused.
type rowScanner interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}
