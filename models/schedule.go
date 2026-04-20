package models

import (
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
	Id            int64     `json:"id,string,omitempty"`
	CronSchedule  string    `json:"cron_schedule,omitempty"`
	AllowedDays   string    `json:"allowed_days,omitempty"`
	AllowedTimes  string    `json:"allowed_times,omitempty"`
	SilenceDays   string    `json:"silence_days,omitempty"`
	SilenceTimes  string    `json:"silence_times,omitempty"`
	ScriptPath    string    `json:"script_path,omitempty"`
	Status        string    `json:"status,omitempty"`
	CreateDate    time.Time `json:"create_date,omitempty"`
}

func validStatus(status string) bool {
	return status == ScheduleStatusEnabled || status == ScheduleStatusDisabled
}

func (s Schedule) All() ([]*Schedule, error) {
	sql := fmt.Sprintf("SELECT `id`, `cron_schedule`, `allowed_days`, `allowed_times`, `silence_days`, `silence_times`, `script_path`, `status`, `create_date` FROM %s", SchedulesTable)
	rows, err := db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schedules := make([]*Schedule, 0)
	for rows.Next() {
		sched := new(Schedule)
		err := rows.Scan(
			&sched.Id,
			&sched.CronSchedule,
			&sched.AllowedDays,
			&sched.AllowedTimes,
			&sched.SilenceDays,
			&sched.SilenceTimes,
			&sched.ScriptPath,
			&sched.Status,
			&sched.CreateDate,
		)
		if err != nil {
			return nil, err
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

	sql := fmt.Sprintf("SELECT `id`, `cron_schedule`, `allowed_days`, `allowed_times`, `silence_days`, `silence_times`, `script_path`, `status`, `create_date` FROM %s WHERE `id`=?", SchedulesTable)
	row := db.QueryRow(sql, id)

	sched := new(Schedule)
	err := row.Scan(
		&sched.Id,
		&sched.CronSchedule,
		&sched.AllowedDays,
		&sched.AllowedTimes,
		&sched.SilenceDays,
		&sched.SilenceTimes,
		&sched.ScriptPath,
		&sched.Status,
		&sched.CreateDate,
	)
	if err != nil {
		return nil, err
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

	var sql string
	if s.Id == 0 {
		s.CreateDate = time.Now()
		s.CreateDate.Format(time.RFC3339)
		sql = fmt.Sprintf("INSERT INTO %s (`cron_schedule`, `allowed_days`, `allowed_times`, `silence_days`, `silence_times`, `script_path`, `status`, `create_date`) VALUES (?,?,?,?,?,?,?,?)", SchedulesTable)
		res, err := db.Exec(sql,
			s.CronSchedule,
			s.AllowedDays,
			s.AllowedTimes,
			s.SilenceDays,
			s.SilenceTimes,
			s.ScriptPath,
			s.Status,
			s.CreateDate,
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

	sql = fmt.Sprintf("UPDATE %s SET `cron_schedule`=?, `allowed_days`=?, `allowed_times`=?, `silence_days`=?, `silence_times`=?, `script_path`=?, `status`=? WHERE `id`=?", SchedulesTable)
	_, err := db.Exec(sql,
		s.CronSchedule,
		s.AllowedDays,
		s.AllowedTimes,
		s.SilenceDays,
		s.SilenceTimes,
		s.ScriptPath,
		s.Status,
		s.Id,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (s Schedule) DeleteById(id int64) error {
	if id < 1 {
		return errors.New("Please provide a valid id")
	}

	sql := fmt.Sprintf("DELETE FROM %s WHERE `id`=?", SchedulesTable)
	_, err := db.Exec(sql, id)
	return err
}
