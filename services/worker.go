package services

import (
	"errors"
	"log"
	"notes/clients"
	"notes/models"
	"os/exec"
	"time"
)

// ErrScheduleNotFound is returned by RunNow when no schedule exists with the
// given id. The routes package maps it to HTTP 404.
var ErrScheduleNotFound = errors.New("schedule not found")

// execScript runs a script_path through /bin/sh -c and returns combined
// output. It is a package-level variable so tests can replace the actual
// shell-out without spawning subprocesses.
var execScript = func(scriptPath string) ([]byte, error) {
	return exec.Command("/bin/sh", "-c", scriptPath).CombinedOutput()
}

// saveAudit persists an audit row. Exposed as a variable so tests can
// substitute an in-memory recorder.
var saveAudit = func(audit models.Audit) (*models.Audit, error) {
	return clients.CreateAudit(audit)
}

// StartWorker launches a background goroutine that ticks at the start of
// every minute, lists all enabled schedules, and executes the script_path
// of any that are due.
func StartWorker() {
	go func() {
		log.Println("[worker] scheduler worker started")

		// Align to the next minute boundary so we tick at :00 each minute.
		now := time.Now().UTC()
		nextMinute := now.Truncate(time.Minute).Add(time.Minute)
		time.Sleep(nextMinute.Sub(now))

		tick := time.NewTicker(time.Minute)
		defer tick.Stop()

		// Run immediately at the first aligned minute, then on every tick.
		runDueSchedules(time.Now().UTC())
		for t := range tick.C {
			runDueSchedules(t.UTC())
		}
	}()
}

func runDueSchedules(t time.Time) {
	enabled, err := Scheduler.ListEnabled()
	if err != nil {
		log.Printf("[worker] error listing schedules: %v", err)
		return
	}
	if len(enabled) == 0 {
		return
	}

	runner := func(sched *models.Schedule) {
		log.Printf("[worker] cron-tick executing schedule id=%d script=%s", sched.Id, sched.ScriptPath)
		_, _ = ExecuteSchedule(sched)
	}

	invoked := Scheduler.InvokeDueAt(enabled, t, runner)
	if len(invoked) > 0 {
		log.Printf("[worker] %d schedule(s) invoked at %s", len(invoked), t.Format("2006-01-02 15:04"))
	}
}

// ExecuteSchedule runs the given schedule's script_path right now, persists
// the resulting audit row, and returns the saved audit. Used by both the
// per-minute worker tick and the on-demand POST /schedules/:id/run handler.
//
// The function never panics on a script failure: a non-zero exit status is
// recorded as an audit with status="failure" and the error string populated;
// the returned error reflects the persistence outcome only (so callers can
// distinguish "script failed but we recorded it" from "we couldn't even
// save the audit row"). When persistence fails the in-memory audit is still
// returned so the HTTP handler can surface the script's output.
func ExecuteSchedule(sched *models.Schedule) (*models.Audit, error) {
	if sched == nil {
		return nil, errors.New("nil schedule")
	}

	startTime := time.Now().UTC()
	out, execErr := execScript(sched.ScriptPath)
	endTime := time.Now().UTC()

	audit := models.Audit{
		ScheduleId: sched.Id,
		ScriptPath: sched.ScriptPath,
		Status:     models.AuditStatusSuccess,
		Output:     string(out),
		StartTime:  startTime,
		EndTime:    endTime,
	}

	if execErr != nil {
		audit.Status = models.AuditStatusFailure
		audit.Error = execErr.Error()
		log.Printf("[worker] schedule id=%d error: %v output: %s", sched.Id, execErr, string(out))
	} else {
		log.Printf("[worker] schedule id=%d finished. output: %s", sched.Id, string(out))
	}

	saved, err := saveAudit(audit)
	if err != nil {
		log.Printf("[worker] schedule id=%d failed to save audit: %v", sched.Id, err)
		return &audit, err
	}
	return saved, nil
}
