package services

import (
	"notes/models"
	"testing"
	"time"
)

// TestInvokeDueAt_TwoSchedulesAtSameTime verifies that out of 5 schedules
// configured with different days and times, the two scheduled at the same
// day/time are the ones invoked when the scheduler is run at that time.
func TestInvokeDueAt_TwoSchedulesAtSameTime(t *testing.T) {
	schedules := []*models.Schedule{
		{
			Id:           1,
			AllowedDays:  "Mon",
			AllowedTimes: "09:00",
			ScriptPath:   "/scripts/a.sh",
			Status:       models.ScheduleStatusEnabled,
		},
		{
			Id:           2,
			AllowedDays:  "Tue",
			AllowedTimes: "10:30",
			ScriptPath:   "/scripts/b.sh",
			Status:       models.ScheduleStatusEnabled,
		},
		{
			Id:           3,
			AllowedDays:  "Wed",
			AllowedTimes: "12:00",
			ScriptPath:   "/scripts/c.sh",
			Status:       models.ScheduleStatusEnabled,
		},
		{
			Id:           4,
			AllowedDays:  "Wed",
			AllowedTimes: "12:00",
			ScriptPath:   "/scripts/d.sh",
			Status:       models.ScheduleStatusEnabled,
		},
		{
			Id:           5,
			AllowedDays:  "Fri",
			AllowedTimes: "17:00",
			ScriptPath:   "/scripts/e.sh",
			Status:       models.ScheduleStatusEnabled,
		},
	}

	// 2024-01-03 is a Wednesday; the only schedules matching Wed @ 12:00
	// are ids 3 and 4.
	target := time.Date(2024, time.January, 3, 12, 0, 0, 0, time.UTC)
	if target.Weekday() != time.Wednesday {
		t.Fatalf("test setup error: expected Wednesday, got %s", target.Weekday())
	}

	invoked := make([]int64, 0, 2)
	result := Scheduler.InvokeDueAt(schedules, target, func(s *models.Schedule) {
		invoked = append(invoked, s.Id)
	})

	if got, want := len(result), 2; got != want {
		t.Fatalf("expected %d schedules to be invoked, got %d (ids=%v)", want, got, invoked)
	}
	if invoked[0] != 3 || invoked[1] != 4 {
		t.Fatalf("expected schedules 3 and 4 to be invoked in order, got %v", invoked)
	}
	for _, s := range result {
		if !IsDueAt(s, target) {
			t.Errorf("schedule %d returned as invoked but IsDueAt=false", s.Id)
		}
	}
}

// TestInvokeDueAt_SkipsDisabled ensures disabled schedules are not invoked
// even when their day/time match.
func TestInvokeDueAt_SkipsDisabled(t *testing.T) {
	schedules := []*models.Schedule{
		{
			Id:           1,
			AllowedDays:  "Wed",
			AllowedTimes: "12:00",
			ScriptPath:   "/scripts/a.sh",
			Status:       models.ScheduleStatusEnabled,
		},
		{
			Id:           2,
			AllowedDays:  "Wed",
			AllowedTimes: "12:00",
			ScriptPath:   "/scripts/b.sh",
			Status:       models.ScheduleStatusDisabled,
		},
	}

	target := time.Date(2024, time.January, 3, 12, 0, 0, 0, time.UTC)
	result := Scheduler.InvokeDueAt(schedules, target, nil)

	if len(result) != 1 || result[0].Id != 1 {
		t.Fatalf("expected only enabled schedule 1 to be invoked, got %+v", result)
	}
}
