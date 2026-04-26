package models

import (
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// withMockDB swaps the package-level db with a sqlmock-backed *sql.DB for the
// duration of the test. The returned cleanup function restores the original
// db, which keeps tests independent.
func withMockDB(t *testing.T) (sqlmock.Sqlmock, func()) {
	t.Helper()
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	orig := db
	db = mockDB
	cleanup := func() {
		db = orig
		_ = mockDB.Close()
	}
	return mock, cleanup
}

// anyTime matches any value of type time.Time. Used so we don't have to know
// the exact CreateDate the model assigns inside Save().
type anyTime struct{}

func (anyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

// TestSchedule_DescriptionRoundTrip verifies that the new Description field
// is honored across the create, update and read paths exposed by the model.
func TestSchedule_DescriptionRoundTrip(t *testing.T) {
	t.Run("create persists description in INSERT", func(t *testing.T) {
		mock, cleanup := withMockDB(t)
		defer cleanup()

		input := Schedule{
			CronSchedule: "0 9 * * 1",
			ScriptPath:   "/scripts/hello.sh",
			Description:  "Weekly Monday reminder",
			Status:       ScheduleStatusEnabled,
		}

		mock.ExpectExec(regexp.QuoteMeta(
			"INSERT INTO schedule (`cron_schedule`, `allowed_days`, `allowed_times`, " +
				"`silence_days`, `silence_times`, `script_path`, `description`, `status`, " +
				"`create_date`, `interval_weeks`, `anchor_date`, `snoozed_until`) " +
				"VALUES (?,?,?,?,?,?,?,?,?,?,?,?)",
		)).WithArgs(
			input.CronSchedule,
			input.AllowedDays,
			input.AllowedTimes,
			input.SilenceDays,
			input.SilenceTimes,
			input.ScriptPath,
			input.Description,
			input.Status,
			anyTime{},
			input.IntervalWeeks,
			nil,                  // anchor date is zero -> NULL
			(*time.Time)(nil),    // snoozed_until is nil
		).WillReturnResult(sqlmock.NewResult(42, 1))

		saved, err := input.Save()
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}
		if saved.Id != 42 {
			t.Errorf("expected Id=42, got %d", saved.Id)
		}
		if saved.Description != input.Description {
			t.Errorf("expected Description=%q, got %q", input.Description, saved.Description)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
	})

	t.Run("update persists description in UPDATE", func(t *testing.T) {
		mock, cleanup := withMockDB(t)
		defer cleanup()

		input := Schedule{
			Id:           7,
			CronSchedule: "0 9 * * 1",
			ScriptPath:   "/scripts/hello.sh",
			Description:  "Updated description text",
			Status:       ScheduleStatusEnabled,
		}

		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE schedule SET `cron_schedule`=?, `allowed_days`=?, `allowed_times`=?, " +
				"`silence_days`=?, `silence_times`=?, `script_path`=?, `description`=?, " +
				"`status`=?, `interval_weeks`=?, `anchor_date`=?, `snoozed_until`=? WHERE `id`=?",
		)).WithArgs(
			input.CronSchedule,
			input.AllowedDays,
			input.AllowedTimes,
			input.SilenceDays,
			input.SilenceTimes,
			input.ScriptPath,
			input.Description,
			input.Status,
			input.IntervalWeeks,
			nil,
			(*time.Time)(nil),
			input.Id,
		).WillReturnResult(sqlmock.NewResult(0, 1))

		saved, err := input.Save()
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}
		if saved.Description != input.Description {
			t.Errorf("expected Description=%q, got %q", input.Description, saved.Description)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
	})

	t.Run("read returns description from SELECT", func(t *testing.T) {
		mock, cleanup := withMockDB(t)
		defer cleanup()

		want := "Read back description"
		createdAt := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)

		columns := []string{
			"id", "cron_schedule", "allowed_days", "allowed_times",
			"silence_days", "silence_times", "script_path", "description",
			"status", "create_date", "interval_weeks", "anchor_date", "snoozed_until",
		}
		rows := sqlmock.NewRows(columns).AddRow(
			int64(7),
			"0 9 * * 1",
			"Mon",
			"09:00",
			"",
			"",
			"/scripts/hello.sh",
			want,
			ScheduleStatusEnabled,
			createdAt,
			1,
			nil,
			nil,
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT `id`, `cron_schedule`, `allowed_days`, `allowed_times`, " +
				"`silence_days`, `silence_times`, `script_path`, `description`, " +
				"`status`, `create_date`, `interval_weeks`, `anchor_date`, `snoozed_until` " +
				"FROM schedule WHERE `id`=?",
		)).WithArgs(int64(7)).WillReturnRows(rows)

		var s Schedule
		got, err := s.FindById(7)
		if err != nil {
			t.Fatalf("FindById returned error: %v", err)
		}
		if got.Description != want {
			t.Errorf("expected Description=%q, got %q", want, got.Description)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
	})
}
