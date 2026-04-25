package services

import (
	"notes/clients"
	"notes/models"
	"strings"
	"time"
)

type SchedulerService struct{}

func NewSchedulerService() *SchedulerService {
	return &SchedulerService{}
}

func (s *SchedulerService) Add(schedule models.Schedule) (*models.Schedule, error) {
	return clients.CreateSchedule(schedule)
}

func (s *SchedulerService) Get(id int64) (*models.Schedule, error) {
	return clients.GetSchedule(id)
}

func (s *SchedulerService) Update(schedule models.Schedule) (*models.Schedule, error) {
	return clients.UpdateSchedule(schedule)
}

func (s *SchedulerService) Delete(id int64) error {
	return clients.DeleteSchedule(id)
}

// Snooze suppresses the schedule with the given id until the provided time.
// The change is persisted via the client layer. The updated schedule is
// returned for convenience.
func (s *SchedulerService) Snooze(id int64, until time.Time) (*models.Schedule, error) {
	sched, err := clients.GetSchedule(id)
	if err != nil {
		return nil, err
	}
	sched.SnoozedUntil = &until
	return clients.UpdateSchedule(*sched)
}

// ClearSnooze removes any active snooze on the schedule with the given id.
func (s *SchedulerService) ClearSnooze(id int64) (*models.Schedule, error) {
	sched, err := clients.GetSchedule(id)
	if err != nil {
		return nil, err
	}
	sched.SnoozedUntil = nil
	return clients.UpdateSchedule(*sched)
}

func (s *SchedulerService) ListEnabled() ([]*models.Schedule, error) {
	all, err := clients.ListSchedules()
	if err != nil {
		return nil, err
	}

	enabled := make([]*models.Schedule, 0, len(all))
	for _, sched := range all {
		if sched.Status == models.ScheduleStatusEnabled {
			enabled = append(enabled, sched)
		}
	}
	return enabled, nil
}

var Scheduler = NewSchedulerService()

// IsDueAt reports whether sched should run at t based on its AllowedDays,
// AllowedTimes, IntervalWeeks/AnchorDate, and SnoozedUntil fields.
//
//   - AllowedDays: comma-separated three-letter weekday names ("Mon,Wed,Fri").
//     Empty matches any day.
//   - AllowedTimes: comma-separated HH:MM times ("09:00,17:30").
//     Empty matches any time.
//   - IntervalWeeks (with AnchorDate): controls recurrence cadence. 0 or 1 means
//     "every week"; N>1 means "every Nth week relative to AnchorDate", and a
//     time before the anchor never matches.
//
// Matching is case-insensitive and tolerant of surrounding whitespace.
func IsDueAt(sched *models.Schedule, t time.Time) bool {
	if sched == nil {
		return false
	}
	if snoozeSuppresses(sched, t) {
		return false
	}
	if !dayMatches(sched.AllowedDays, t) {
		return false
	}
	if !timeMatches(sched.AllowedTimes, t) {
		return false
	}
	if !intervalMatches(sched, t) {
		return false
	}
	return true
}

// snoozeSuppresses reports whether sched is currently snoozed past t.
func snoozeSuppresses(sched *models.Schedule, t time.Time) bool {
	return sched.SnoozedUntil != nil && t.Before(*sched.SnoozedUntil)
}

func intervalMatches(sched *models.Schedule, t time.Time) bool {
	interval := sched.IntervalWeeks
	if interval <= 1 {
		return true
	}
	if sched.AnchorDate.IsZero() {
		// Cadence requested but no anchor — treat as every week to avoid
		// silently dropping firings.
		return true
	}
	days := daysBetween(sched.AnchorDate, t)
	if days < 0 {
		return false
	}
	return (days/7)%interval == 0
}

// daysBetween returns the number of whole calendar days from a to b, comparing
// only the date components in UTC. Negative if b is before a.
func daysBetween(a, b time.Time) int {
	aDay := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	bDay := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return int(bDay.Sub(aDay) / (24 * time.Hour))
}

func dayMatches(allowedDays string, t time.Time) bool {
	allowedDays = strings.TrimSpace(allowedDays)
	if allowedDays == "" {
		return true
	}
	want := strings.ToLower(t.Weekday().String()[:3])
	for _, part := range strings.Split(allowedDays, ",") {
		if strings.ToLower(strings.TrimSpace(part)) == want {
			return true
		}
	}
	return false
}

func timeMatches(allowedTimes string, t time.Time) bool {
	allowedTimes = strings.TrimSpace(allowedTimes)
	if allowedTimes == "" {
		return true
	}
	want := t.Format("15:04")
	for _, part := range strings.Split(allowedTimes, ",") {
		if strings.TrimSpace(part) == want {
			return true
		}
	}
	return false
}

// InvokeDueAt runs runner for every enabled schedule in schedules whose
// AllowedDays/AllowedTimes match t, preserving input order. The slice of
// invoked schedules is returned for inspection.
func (s *SchedulerService) InvokeDueAt(schedules []*models.Schedule, t time.Time, runner func(*models.Schedule)) []*models.Schedule {
	invoked := make([]*models.Schedule, 0, len(schedules))
	for _, sched := range schedules {
		if sched == nil || sched.Status != models.ScheduleStatusEnabled {
			continue
		}
		if !IsDueAt(sched, t) {
			continue
		}
		if runner != nil {
			runner(sched)
		}
		invoked = append(invoked, sched)
	}
	return invoked
}
