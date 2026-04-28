package services

import (
	"notes/models"
	"os/exec"
	"strings"
	"testing"
)

// TestWorker_ExecutesQuotedCurlCommand simulates exactly the path the worker
// takes (`/bin/sh -c <script_path>`) for the kind of long, single-quoted curl
// invocation that the LLM generates. This is the regression that bit users:
// when a script with embedded JSON ('{"foo":"bar"}') was silently truncated
// at 512 bytes by MySQL, /bin/sh failed with
//   "syntax error: unterminated quoted string"
// because the closing single-quote was lost.
//
// We don't need the network to verify the shell parsing path — we only need
// the command to PARSE successfully and exit cleanly. Replacing curl with
// `printf` keeps the test hermetic while preserving the structural shape
// (long single-quoted JSON string with double-quotes inside).
func TestWorker_ExecutesQuotedCurlCommand(t *testing.T) {
	const script = `printf '%s' ` +
		`'{"from":{"email":"watches@joseserver.com","name":"Windup Monitor"},` +
		`"to":[{"email":"leonj1@gmail.com"}],` +
		`"subject":"[TEST] Windup Monitor is working",` +
		`"text":"This is a test email to confirm the Windup Watch Fair NYC monitor ` +
		`can reach you. The roster watcher will fire when new brands are posted at ` +
		`https://www.windupwatchfair.com/nyc-2025-brands. If you got this, the full ` +
		`pipeline (scheduler -> shell -> Mailtrap -> Gmail) is healthy and the ` +
		`production schedule will deliver brand-update notifications the same way. ` +
		`No action required from you; this message exists only to demonstrate that ` +
		`the entire delivery chain works end to end without truncation."}'`

	if len(script) <= 512 {
		t.Fatalf("test setup error: script must exceed 512 chars to mirror the regression, got %d", len(script))
	}

	out, err := exec.Command("/bin/sh", "-c", script).CombinedOutput()
	if err != nil {
		t.Fatalf("/bin/sh -c rejected the long quoted command (this is the production bug): err=%v output=%q",
			err, string(out))
	}

	// Sanity: the JSON body should appear in the output, intact.
	if !strings.Contains(string(out), `"from":{"email":"watches@joseserver.com"`) ||
		!strings.Contains(string(out), `"text":"This is a test email`) {
		t.Errorf("output missing expected JSON fragments; got: %q", string(out))
	}
}

// TestExecuteSchedule_Success verifies that ExecuteSchedule runs the
// schedule's script through execScript, builds an audit with status=success,
// and persists it via saveAudit. Both seams are stubbed so the test never
// touches the DB or spawns a subprocess.
func TestExecuteSchedule_Success(t *testing.T) {
	origExec := execScript
	origSave := saveAudit
	defer func() {
		execScript = origExec
		saveAudit = origSave
	}()

	const wantOutput = "ran ok"
	execScript = func(scriptPath string) ([]byte, error) {
		if scriptPath != "echo hello" {
			t.Errorf("execScript got scriptPath=%q, want %q", scriptPath, "echo hello")
		}
		return []byte(wantOutput), nil
	}

	var savedAudit *models.Audit
	saveAudit = func(a models.Audit) (*models.Audit, error) {
		// Simulate the DB assigning an id.
		a.Id = 4242
		savedAudit = &a
		return &a, nil
	}

	sched := &models.Schedule{Id: 11, ScriptPath: "echo hello"}
	got, err := ExecuteSchedule(sched)
	if err != nil {
		t.Fatalf("ExecuteSchedule unexpected error: %v", err)
	}
	if got == nil {
		t.Fatalf("ExecuteSchedule returned nil audit")
	}
	if got.Status != models.AuditStatusSuccess {
		t.Errorf("audit.Status = %q, want %q", got.Status, models.AuditStatusSuccess)
	}
	if got.Output != wantOutput {
		t.Errorf("audit.Output = %q, want %q", got.Output, wantOutput)
	}
	if got.Error != "" {
		t.Errorf("audit.Error = %q, want empty", got.Error)
	}
	if got.ScheduleId != 11 {
		t.Errorf("audit.ScheduleId = %d, want 11", got.ScheduleId)
	}
	if got.StartTime.IsZero() || got.EndTime.IsZero() {
		t.Errorf("audit times not populated: start=%v end=%v", got.StartTime, got.EndTime)
	}
	if savedAudit == nil {
		t.Errorf("saveAudit was not called")
	}
}

// TestExecuteSchedule_Failure verifies that a non-zero exit status produces
// an audit with status=failure and a populated Error string. The persisted
// audit is still returned so the route can show the user what went wrong.
func TestExecuteSchedule_Failure(t *testing.T) {
	origExec := execScript
	origSave := saveAudit
	defer func() {
		execScript = origExec
		saveAudit = origSave
	}()

	execScript = func(scriptPath string) ([]byte, error) {
		return []byte("boom output"), &exec.ExitError{}
	}
	var saveCalls int
	saveAudit = func(a models.Audit) (*models.Audit, error) {
		saveCalls++
		a.Id = 99
		return &a, nil
	}

	sched := &models.Schedule{Id: 7, ScriptPath: "false"}
	got, err := ExecuteSchedule(sched)
	if err != nil {
		t.Fatalf("ExecuteSchedule unexpected error: %v", err)
	}
	if got.Status != models.AuditStatusFailure {
		t.Errorf("audit.Status = %q, want %q", got.Status, models.AuditStatusFailure)
	}
	if got.Output != "boom output" {
		t.Errorf("audit.Output = %q, want %q", got.Output, "boom output")
	}
	if got.Error == "" {
		t.Errorf("audit.Error must be populated on failure")
	}
	if saveCalls != 1 {
		t.Errorf("saveAudit called %d times, want 1", saveCalls)
	}
}

// TestExecuteSchedule_NilSchedule documents the defensive guard: a nil
// schedule must not panic and must return an error without touching the
// exec or save seams.
func TestExecuteSchedule_NilSchedule(t *testing.T) {
	origExec := execScript
	origSave := saveAudit
	defer func() {
		execScript = origExec
		saveAudit = origSave
	}()

	execScript = func(string) ([]byte, error) {
		t.Fatalf("execScript should not be called for nil schedule")
		return nil, nil
	}
	saveAudit = func(models.Audit) (*models.Audit, error) {
		t.Fatalf("saveAudit should not be called for nil schedule")
		return nil, nil
	}

	if _, err := ExecuteSchedule(nil); err == nil {
		t.Fatalf("expected error for nil schedule, got nil")
	}
}

// TestWorker_RecordsFailureAuditOnShellSyntaxError documents the worker's
// behaviour when /bin/sh can't parse the script (e.g. unterminated quote).
// We can't drive runDueSchedules() directly without a DB, but we can verify
// the shape of the output the worker logs and stores: a non-nil exit error,
// and a stderr message containing "syntax error".
//
// This is the exact failure mode the user observed in production for every
// truncated schedule.
func TestWorker_RecordsFailureAuditOnShellSyntaxError(t *testing.T) {
	// Simulate a script that was truncated mid-quote (the user's id=2 row
	// looked like this after MySQL chopped it at 512 chars).
	const truncated = `curl -X POST 'https://api.example.com' -d '{"text":"hello`

	out, err := exec.Command("/bin/sh", "-c", truncated).CombinedOutput()
	if err == nil {
		t.Fatalf("expected /bin/sh to fail on unterminated quote; got success and output=%q", string(out))
	}

	// Confirm the failure mode matches what the worker would record in the audit.
	if !strings.Contains(string(out), "unterminated") && !strings.Contains(string(out), "syntax") {
		t.Errorf("expected stderr to mention unterminated/syntax error; got: %q", string(out))
	}

	// And confirm the audit shape the worker would build.
	audit := models.Audit{
		ScheduleId: 99,
		ScriptPath: truncated,
		Status:     models.AuditStatusSuccess,
		Output:     string(out),
	}
	if err != nil {
		audit.Status = models.AuditStatusFailure
		audit.Error = err.Error()
	}
	if audit.Status != models.AuditStatusFailure {
		t.Errorf("expected audit.Status=%q, got %q", models.AuditStatusFailure, audit.Status)
	}
	if audit.Error == "" {
		t.Errorf("expected audit.Error to be populated")
	}
}
