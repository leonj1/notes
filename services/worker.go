package services

import (
	"log"
	"notes/clients"
	"notes/models"
	"os/exec"
	"time"
)

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
		log.Printf("[worker] executing schedule id=%d script=%s", sched.Id, sched.ScriptPath)
		startTime := time.Now().UTC()
		out, execErr := exec.Command("/bin/sh", "-c", sched.ScriptPath).CombinedOutput()
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

		if _, err := clients.CreateAudit(audit); err != nil {
			log.Printf("[worker] schedule id=%d failed to save audit: %v", sched.Id, err)
		}
	}

	invoked := Scheduler.InvokeDueAt(enabled, t, runner)
	if len(invoked) > 0 {
		log.Printf("[worker] %d schedule(s) invoked at %s", len(invoked), t.Format("2006-01-02 15:04"))
	}
}
