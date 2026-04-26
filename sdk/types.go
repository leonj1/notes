package sdk

import "time"

// ScheduleStatus enumerates the values accepted by Schedule.Status.
const (
	ScheduleStatusEnabled  = "enabled"
	ScheduleStatusDisabled = "disabled"
)

// AuditStatus enumerates the values produced by the server in Audit.Status.
const (
	AuditStatusSuccess = "success"
	AuditStatusFailure = "failure"
)

// Schedule mirrors the JSON contract exposed by the notes service for the
// /schedules endpoints. Field tags match the server (snake_case, with int64
// IDs encoded as strings).
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
	IntervalWeeks int        `json:"interval_weeks,omitempty"`
	AnchorDate    time.Time  `json:"anchor_date,omitempty"`
	SnoozedUntil  *time.Time `json:"snoozed_until,omitempty"`
}

// Audit mirrors the JSON contract exposed by the notes service for the
// /audits endpoints.
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
