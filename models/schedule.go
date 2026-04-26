package models

import (
	"database/sql"
	"fmt"
	"github.com/kataras/go-errors"
	"time"
)

const SchedulesTable = "schedule"

const (
	ScheduleStatusEnabled  = "enabled"
	ScheduleStatusDisabled = "disabled"
)

type Schedule struct {
	Id            int64      `json:"id,string,omitempty"`
	CronSchedule  string     `json:"cron_schedule,omitempty"`
	AllowedDays   string     `json:"allowed_days,omitempty"`
	AllowedTimes  string     `json:"allowed_times,omitempty"`
	SilenceDays   string     `json:"silence_days,omitempty"`
	SilenceTimes  string     `json:"silence_times,omitempty"`
	ScriptPath    string     `json:"script_path,omitempty"`
	Description   string     `json:"description,omitempty"`
	Status        string     `json:"status,omitempty"`
	CreateDate    time.Time  `json:"create_date,omitempty"`
	// IntervalWeeks controls recurrence cadence. 0 or 1 means "every week";
	// N means "every Nth week relative to AnchorDate".
	IntervalWeeks int        `json:"interval_weeks,omitempty"`
	// AnchorDate is the reference date used to compute week parity when
	// IntervalWeeks > 1. If zero, IntervalWeeks is ignored.
	AnchorDate    time.Time  `json:"anchor_date,omitempty"`
	// SnoozedUntil suppresses the schedule for any firing time before it.
	// Nil means "not snoozed".
	SnoozedUntil  *time.Time `json:"snoozed_until,omitempty"`
}

func validStatus(status string) bool {
	return status == ScheduleStatusEnabled || status == ScheduleStatusDisabled
}

func (s Schedule) All() ([]*Schedule, error) {
	query := fmt.Sprintf("SELECT `id`, `cron_schedule`, `allowed_days`, `allowed_times`, `silence_days`, `silence_times`, `script_path`, `description`, `status`, `create_date`, `interval_weeks`, `anchor_date`, `snoozed_until` FROM %s", SchedulesTable)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schedules := make([]*Schedule, 0)
	for rows.Next() {
		sched := new(Schedule)
		var anchor sql.NullTime
		var snoozed sql.NullTime
		err := rows.Scan(
			&sched.Id,
			&sched.CronSchedule,
			&sched.AllowedDays,
			&sched.AllowedTimes,
			&sched.SilenceDays,
			&sched.SilenceTimes,
			&sched.ScriptPath,
			&sched.Description,
			&sched.Status,
			&sched.CreateDate,
			&sched.IntervalWeeks,
			&anchor,
			&snoozed,
		)
		if err != nil {
			return nil, err
		}
		if anchor.Valid {
			sched.AnchorDate = anchor.Time
		}
		if snoozed.Valid {
			t := snoozed.Time
			sched.SnoozedUntil = &t
		}
		schedules = append(schedules, sched)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return schedules, nil
}

func (s Schedule) FindById(id int64) (*Schedule, error) {
	if id < 1 {
		return nil, errors.New("Please provide a valid id")
	}

	query := fmt.Sprintf("SELECT `id`, `cron_schedule`, `allowed_days`, `allowed_times`, `silence_days`, `silence_times`, `script_path`, `description`, `status`, `create_date`, `interval_weeks`, `anchor_date`, `snoozed_until` FROM %s WHERE `id`=?", SchedulesTable)
	row := db.QueryRow(query, id)

	sched := new(Schedule)
	var anchor sql.NullTime
	var snoozed sql.NullTime
	err := row.Scan(
		&sched.Id,
		&sched.CronSchedule,
		&sched.AllowedDays,
		&sched.AllowedTimes,
		&sched.SilenceDays,
		&sched.SilenceTimes,
		&sched.ScriptPath,
		&sched.Description,
		&sched.Status,
		&sched.CreateDate,
		&sched.IntervalWeeks,
		&anchor,
		&snoozed,
	)
	if err != nil {
		return nil, err
	}
	if anchor.Valid {
		sched.AnchorDate = anchor.Time
	}
	if snoozed.Valid {
		t := snoozed.Time
		sched.SnoozedUntil = &t
	}
	return sched, nil
}

func (s Schedule) Save() (*Schedule, error) {
	if s.CronSchedule == "" || s.ScriptPath == "" {
		return nil, errors.New("cron_schedule and script_path are required")
	}
	if s.Status == "" {
		s.Status = ScheduleStatusDisabled
	}
	if !validStatus(s.Status) {
		return nil, errors.New("status must be 'enabled' or 'disabled'")
	}

	var query string
	if s.Id == 0 {
		s.CreateDate = time.Now()
		s.CreateDate.Format(time.RFC3339)
		query = fmt.Sprintf("INSERT INTO %s (`cron_schedule`, `allowed_days`, `allowed_times`, `silence_days`, `silence_times`, `script_path`, `description`, `status`, `create_date`, `interval_weeks`, `anchor_date`, `snoozed_until`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)", SchedulesTable)
		res, err := db.Exec(query,
			s.CronSchedule,
			s.AllowedDays,
			s.AllowedTimes,
			s.SilenceDays,
			s.SilenceTimes,
			s.ScriptPath,
			s.Description,
			s.Status,
			s.CreateDate,
			s.IntervalWeeks,
			nullableTime(s.AnchorDate),
			s.SnoozedUntil,
		)
		if err != nil {
			return nil, err
		}
		s.Id, err = res.LastInsertId()
		if err != nil {
			return nil, err
		}
		return &s, nil
	}

	query = fmt.Sprintf("UPDATE %s SET `cron_schedule`=?, `allowed_days`=?, `allowed_times`=?, `silence_days`=?, `silence_times`=?, `script_path`=?, `description`=?, `status`=?, `interval_weeks`=?, `anchor_date`=?, `snoozed_until`=? WHERE `id`=?", SchedulesTable)
	_, err := db.Exec(query,
		s.CronSchedule,
		s.AllowedDays,
		s.AllowedTimes,
		s.SilenceDays,
		s.SilenceTimes,
		s.ScriptPath,
		s.Description,
		s.Status,
		s.IntervalWeeks,
		nullableTime(s.AnchorDate),
		s.SnoozedUntil,
		s.Id,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// nullableTime converts a zero time to a SQL NULL and a non-zero time to a
// valid sql.NullTime. Used so the model can persist optional dates without
// switching the field type away from time.Time.
func nullableTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func (s Schedule) DeleteById(id int64) error {
	if id < 1 {
		return errors.New("Please provide a valid id")
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE `id`=?", SchedulesTable)
	_, err := db.Exec(query, id)
	return err
}
