package services

import (
	"notes/models"
	"testing"
	"time"
)

// ts is a small helper for building UTC timestamps in tests.
func ts(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
}

// TestIsDueAt drives the matcher with table-driven, BDD-style scenarios.
// Each case is one Given/When/Then sentence that runs in its own subtest.
func TestIsDueAt(t *testing.T) {
	monAt0900 := models.Schedule{
		AllowedDays:  "Mon",
		AllowedTimes: "09:00",
		Status:       models.ScheduleStatusEnabled,
	}
	biweeklyFri := models.Schedule{
		AllowedDays:   "Fri",
		AllowedTimes:  "17:00",
		IntervalWeeks: 2,
		AnchorDate:    ts(2024, 4, 26, 0, 0), // 2024-04-26 is a Friday
		Status:        models.ScheduleStatusEnabled,
	}
	snoozeUntil := ts(2024, 4, 27, 12, 0) // Saturday noon
	weekendSnoozedTilSatNoon := models.Schedule{
		AllowedDays:  "Fri,Sat,Sun",
		AllowedTimes: "08:00",
		Status:       models.ScheduleStatusEnabled,
		SnoozedUntil: &snoozeUntil,
	}
	weekendReminder := models.Schedule{
		AllowedDays:  "Fri,Sat,Sun",
		AllowedTimes: "08:00",
		Status:       models.ScheduleStatusEnabled,
	}

	cases := []struct {
		name string
		s    models.Schedule
		t    time.Time
		want bool
	}{
		{
			name: "given Mon 09:00 schedule, when Mon at 09:00, then due",
			s:    monAt0900,
			t:    ts(2024, 4, 22, 9, 0), // 2024-04-22 is a Monday
			want: true,
		},
		{
			name: "given Mon 09:00 schedule, when Tue at 09:00, then not due",
			s:    monAt0900,
			t:    ts(2024, 4, 23, 9, 0),
			want: false,
		},
		{
			name: "given Mon 09:00 schedule, when Mon at 09:01, then not due",
			s:    monAt0900,
			t:    ts(2024, 4, 22, 9, 1),
			want: false,
		},
		{
			name: "given biweekly Fri 17:00 schedule, when anchor Friday at 17:00, then due",
			s:    biweeklyFri,
			t:    ts(2024, 4, 26, 17, 0),
			want: true,
		},
		{
			name: "given biweekly Fri 17:00 schedule, when off-week Friday at 17:00, then not due",
			s:    biweeklyFri,
			t:    ts(2024, 5, 3, 17, 0),
			want: false,
		},
		{
			name: "given biweekly Fri 17:00 schedule, when next on-week Friday at 17:00, then due",
			s:    biweeklyFri,
			t:    ts(2024, 5, 10, 17, 0),
			want: true,
		},
		{
			name: "given biweekly Fri 17:00 schedule, when before anchor, then not due",
			s:    biweeklyFri,
			t:    ts(2024, 4, 19, 17, 0),
			want: false,
		},
		{
			name: "given weekend reminder snoozed until Sat noon, when Sat at 08:00, then suppressed",
			s:    weekendSnoozedTilSatNoon,
			t:    ts(2024, 4, 27, 8, 0),
			want: false,
		},
		{
			name: "given weekend reminder snoozed until Sat noon, when Sun at 08:00, then due",
			s:    weekendSnoozedTilSatNoon,
			t:    ts(2024, 4, 28, 8, 0),
			want: true,
		},
		{
			name: "given weekend reminder snoozed until Sat noon, when Sat at 13:00 (past snooze, off-schedule time), then not due",
			s:    weekendSnoozedTilSatNoon,
			t:    ts(2024, 4, 27, 13, 0),
			want: false,
		},
		{
			name: "given single weekend reminder Fri/Sat/Sun 08:00, when Friday at 08:00, then due",
			s:    weekendReminder,
			t:    ts(2024, 4, 26, 8, 0),
			want: true,
		},
		{
			name: "given single weekend reminder Fri/Sat/Sun 08:00, when Saturday at 08:00, then due",
			s:    weekendReminder,
			t:    ts(2024, 4, 27, 8, 0),
			want: true,
		},
		{
			name: "given single weekend reminder Fri/Sat/Sun 08:00, when Sunday at 08:00, then due",
			s:    weekendReminder,
			t:    ts(2024, 4, 28, 8, 0),
			want: true,
		},
		{
			name: "given single weekend reminder Fri/Sat/Sun 08:00, when Monday at 08:00, then not due",
			s:    weekendReminder,
			t:    ts(2024, 4, 29, 8, 0),
			want: false,
		},
	}

	for _, tc := range cases {
		s := tc.s
		t.Run(tc.name, func(t *testing.T) {
			if got := IsDueAt(&s, tc.t); got != tc.want {
				t.Fatalf("IsDueAt = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestInvokeDueAt_TwoSchedulesAtSameTime verifies that out of 5 schedules
// configured with different days and times, the two scheduled at the same
// day/time are the ones invoked when the scheduler is run at that time.
func TestInvokeDueAt_TwoSchedulesAtSameTime(t *testing.T) {
	schedules := []*models.Schedule{
		{Id: 1, AllowedDays: "Mon", AllowedTimes: "09:00", Status: models.ScheduleStatusEnabled},
		{Id: 2, AllowedDays: "Tue", AllowedTimes: "10:30", Status: models.ScheduleStatusEnabled},
		{Id: 3, AllowedDays: "Wed", AllowedTimes: "12:00", Status: models.ScheduleStatusEnabled},
		{Id: 4, AllowedDays: "Wed", AllowedTimes: "12:00", Status: models.ScheduleStatusEnabled},
		{Id: 5, AllowedDays: "Fri", AllowedTimes: "17:00", Status: models.ScheduleStatusEnabled},
	}

	target := ts(2024, 1, 3, 12, 0) // 2024-01-03 is a Wednesday
	if target.Weekday() != time.Wednesday {
		t.Fatalf("test setup error: expected Wednesday, got %s", target.Weekday())
	}

	invoked := make([]int64, 0, 2)
	result := Scheduler.InvokeDueAt(schedules, target, func(s *models.Schedule) {
		invoked = append(invoked, s.Id)
	})

	if got, want := len(result), 2; got != want {
		t.Fatalf("expected %d schedules invoked, got %d (ids=%v)", want, got, invoked)
	}
	if invoked[0] != 3 || invoked[1] != 4 {
		t.Fatalf("expected schedules 3 and 4 in order, got %v", invoked)
	}
}

// TestInvokeDueAt_SkipsDisabled ensures disabled schedules are not invoked
// even when their day/time match.
func TestInvokeDueAt_SkipsDisabled(t *testing.T) {
	schedules := []*models.Schedule{
		{Id: 1, AllowedDays: "Wed", AllowedTimes: "12:00", Status: models.ScheduleStatusEnabled},
		{Id: 2, AllowedDays: "Wed", AllowedTimes: "12:00", Status: models.ScheduleStatusDisabled},
	}

	target := ts(2024, 1, 3, 12, 0)
	result := Scheduler.InvokeDueAt(schedules, target, nil)

	if len(result) != 1 || result[0].Id != 1 {
		t.Fatalf("expected only enabled schedule 1, got %+v", result)
	}
}
