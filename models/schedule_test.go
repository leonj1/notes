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

// TestSchedule_LongScriptPathRoundTrip is a regression test for the silent
// MySQL truncation that surfaced in production: schedule.script_path used to
// be VARCHAR(512), and because the docker-compose stack runs MySQL with
// --sql-mode=NO_ENGINE_SUBSTITUTION (NOT strict), oversize values were
// silently chopped at 512 chars, leaving curl invocations with unterminated
// quoted strings. V4__schedule_script_path_text.sql widens the column to
// TEXT to fix this; this test asserts the model layer itself doesn't impose
// a length cap on the way in or out, so the column type is the only place
// truncation could possibly happen.
//
// The script body used here (>1KB) mirrors what the LLM builds when the
// schedule_create tool is asked to fire a curl POST with a JSON body.
func TestSchedule_LongScriptPathRoundTrip(t *testing.T) {
	const longScript = `[ -f /tmp/ww-nyc-test-sent ] && exit 0; ` +
		`curl -fsS -X POST 'https://send.api.mailtrap.io/api/send' ` +
		`-H 'Authorization: Bearer 5c7508f840e5bf0ffba66f0d92a69e36' ` +
		`-H 'Content-Type: application/json' ` +
		`-d '{"from":{"email":"watches@joseserver.com","name":"Windup Monitor"},` +
		`"to":[{"email":"leonj1@gmail.com"}],` +
		`"subject":"[TEST] Windup Monitor is working",` +
		`"text":"This is a test email to confirm the Windup Watch Fair NYC monitor ` +
		`can reach you. The roster watcher will fire when new brands are posted at ` +
		`https://www.windupwatchfair.com/nyc-2025-brands. If you got this, the full ` +
		`pipeline (scheduler -> shell -> Mailtrap -> Gmail) is healthy and the ` +
		`production schedule will deliver brand-update notifications the same way. ` +
		`No action required."}' && touch /tmp/ww-nyc-test-sent`

	if len(longScript) <= 512 {
		t.Fatalf("test setup error: script must exceed 512 chars to exercise the regression, got %d", len(longScript))
	}

	t.Run("create persists the entire long script_path", func(t *testing.T) {
		mock, cleanup := withMockDB(t)
		defer cleanup()

		input := Schedule{
			CronSchedule: "* * * * *",
			ScriptPath:   longScript,
			Description:  "Mailtrap test ping",
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
			longScript, // must be passed unmodified to the driver
			input.Description,
			input.Status,
			anyTime{},
			input.IntervalWeeks,
			nil,
			(*time.Time)(nil),
		).WillReturnResult(sqlmock.NewResult(101, 1))

		saved, err := input.Save()
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}
		if saved.ScriptPath != longScript {
			t.Errorf("expected ScriptPath of length %d preserved, got length %d",
				len(longScript), len(saved.ScriptPath))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
	})

	t.Run("read returns the entire long script_path from SELECT", func(t *testing.T) {
		mock, cleanup := withMockDB(t)
		defer cleanup()

		columns := []string{
			"id", "cron_schedule", "allowed_days", "allowed_times",
			"silence_days", "silence_times", "script_path", "description",
			"status", "create_date", "interval_weeks", "anchor_date", "snoozed_until",
		}
		rows := sqlmock.NewRows(columns).AddRow(
			int64(101),
			"* * * * *",
			"",
			"",
			"",
			"",
			longScript,
			"Mailtrap test ping",
			ScheduleStatusEnabled,
			time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC),
			1,
			nil,
			nil,
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT `id`, `cron_schedule`, `allowed_days`, `allowed_times`, " +
				"`silence_days`, `silence_times`, `script_path`, `description`, " +
				"`status`, `create_date`, `interval_weeks`, `anchor_date`, `snoozed_until` " +
				"FROM schedule WHERE `id`=?",
		)).WithArgs(int64(101)).WillReturnRows(rows)

		var s Schedule
		got, err := s.FindById(101)
		if err != nil {
			t.Fatalf("FindById returned error: %v", err)
		}
		if got.ScriptPath != longScript {
			t.Errorf("expected ScriptPath of length %d, got length %d",
				len(longScript), len(got.ScriptPath))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
	})
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
