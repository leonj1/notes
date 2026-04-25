# Plan: On-Demand Invocation of Scheduled Jobs (Async)

## 1. Objective

Allow API clients to trigger an existing schedule's `script_path` immediately,
without waiting for the time-based worker to find it due. Because script
duration is unbounded, the API is **asynchronous**: the trigger call returns
a `run_id` immediately, and the caller polls a status endpoint to observe
progress and final outcome. The execution must reuse the same code path the
worker uses today so behavior, logging, and audit records stay consistent.

## 2. Background (Current State)

- The HTTP surface for schedules in `notes.go:46-49` exposes only CRUD:
  `POST/GET /schedules`, `GET/DELETE /schedules/:id`.
- `services/worker.go:14-32` is the only caller that actually runs a
  schedule's script. It ticks every minute and calls
  `Scheduler.InvokeDueAt`, which gates execution on `IsDueAt`
  (`services/scheduler.go:83-100`).
- The execution + audit logic lives inline as a closure inside
  `runDueSchedules` (`services/worker.go:44-70`); it is not reachable from
  HTTP handlers today.
- The audit row is written **once, after** the script finishes, via
  `clients.CreateAudit` (`services/worker.go:67`). There is no concept of
  an in-flight run; the row only exists post-mortem.
- `models.Audit` has only `success` / `failure` statuses
  (`models/audit.go:11-14`) and `start_time` / `end_time` are both
  `NOT NULL` in the schema (`migrations/V2__schedule_audit.sql:11-12`).

## 3. Scope

### In scope

- New HTTP endpoint to *enqueue* a single schedule on demand by id.
- New HTTP endpoint to fetch the status / result of a run by `run_id`.
- Refactor the script-execution + audit-write logic out of the worker
  closure into a reusable service method that supports an in-flight
  lifecycle (insert `running`, then update to terminal status).
- Wire the worker to call the new method.
- Persist an audit row for every run, distinguishable as `manual` vs
  `scheduled` and queryable while still in flight.
- Unit tests for the service method and the new route handlers.

### Out of scope

- Cancelling an in-flight run (no SIGTERM endpoint).
- Streaming script output to the client (poll-only for v1).
- Authentication / authorization (project has none today).
- Distributed locking across multiple `notes` processes — only single-process
  guarantees are made.
- Any UI changes (project is API-only).
- Webhook / push notification on completion.

## 4. Design Decisions

### 4.1 Async lifecycle and the `run_id` contract

The `run_id` returned to clients is the **`audit.id`** of the row created
the moment the run is enqueued. This means:

- One row per run, present from the instant the run starts (not just at
  the end as today).
- The audit row carries the live state machine of the run, so polling for
  status is just `GET` on that row.
- No second table is needed.

Lifecycle of a single run:

```
client                 server                       audit row
------                 ------                       ---------
POST .../run    -->    validate + load schedule
                       INSERT audit (status=running, start_time=now,
                                     end_time=NULL, trigger=manual,
                                     output='', error='')
                       go func() { exec script ... }
                <--    202 Accepted { run_id, status: "running", ... }

GET /runs/:id   -->    SELECT audit WHERE id=:id
                <--    200 OK { ...current state... }

(later, in goroutine)
                       exec finishes
                       UPDATE audit SET status='success'|'failure',
                                        output=..., error=...,
                                        end_time=now
                                 WHERE id=run_id

GET /runs/:id   -->    SELECT audit WHERE id=:id
                <--    200 OK { status: "success", output: "...", end_time: ... }
```

### 4.2 Endpoint shapes

#### 4.2.1 `POST /schedules/:id/run` — enqueue a run

- Request body: empty for v1. Future option: JSON overrides like
  `{ "respect_status": true }` without breaking the route.
- Response: `202 Accepted` with the freshly-inserted audit JSON. The
  client reads `id` from the body and uses it as `run_id`. A
  `Location: /runs/<run_id>` header is also set so clients that follow
  Location can poll directly.
- Status codes:
  - `202 Accepted` — run successfully enqueued (script may still be
    starting or running).
  - `400 Bad Request` — invalid id parameter.
  - `404 Not Found` — schedule id does not exist.
  - `500 Internal Server Error` — DB / persistence failure while writing
    the initial `running` row, or schedule lookup failure other than not
    found. Parity with existing handlers
    (e.g. `routes/schedules.go:48-51`).

The handler returns *before* the script starts executing, so a slow
script never blocks the HTTP request.

#### 4.2.2 `GET /runs/:id` — fetch status / result

- Path style mirrors how runs are first-class objects, not just a
  sub-resource of audits. Since the audit row *is* the run, this is
  effectively `GET /audits/:id`; we register **both** as aliases for
  the same handler so existing audit consumers stay happy and new
  callers get a semantic name.
- Response: `200 OK` with the full `Audit` JSON (`status` is one of
  `running`, `success`, `failure`).
- Status codes:
  - `200 OK` — found.
  - `400 Bad Request` — invalid id parameter.
  - `404 Not Found` — no audit / run with that id.
  - `500 Internal Server Error` — DB error.

Polling cadence: client choice; for v1 we recommend 1–5 seconds in docs.

### 4.3 What rules does on-demand execution bypass?

On-demand is a deliberate human-driven action and should bypass time-based
gates so it is actually useful:

| Rule                          | Worker today | On-demand     |
|-------------------------------|--------------|---------------|
| `AllowedDays` / `AllowedTimes`| Enforced     | **Bypass**    |
| `IntervalWeeks` / `AnchorDate`| Enforced     | **Bypass**    |
| `SnoozedUntil`                | Enforced     | **Bypass**    |
| `Status == disabled`          | Skips run    | **Bypass**    |

Rationale: snooze and disable are protections against *automatic* firing.
A manual `POST .../run` is an explicit override; refusing to honor it would
make the endpoint nearly useless. Documented in code comments.

If we later need a "respect status" mode, add a query / body flag
(`?respect_status=true`); not in v1.

### 4.4 Audit / run model changes

#### 4.4.1 New status

Add `running` alongside `success` / `failure`
(`models/audit.go:11-14`):

```go
const (
    AuditStatusRunning = "running"
    AuditStatusSuccess = "success"
    AuditStatusFailure = "failure"
)
```

Terminal states stay `success` / `failure`. `running` is the only
non-terminal state for v1.

#### 4.4.2 New trigger column

Distinguish manual from scheduled:

```go
const (
    AuditTriggerScheduled = "scheduled"
    AuditTriggerManual    = "manual"
)
```

Stored as `trigger_type` (SQL — `TRIGGER` is reserved) but exposed as
`trigger` in JSON for readability.

#### 4.4.3 `end_time` becomes nullable

Today `end_time` is `NOT NULL` (`migrations/V2__schedule_audit.sql:12`).
For an in-flight `running` row we don't have an end yet, so the column
must allow NULL. Go side: switch to `sql.NullTime` and expose a pointer
in JSON so it serializes as `null` while running.

#### 4.4.4 Migration `migrations/V3__schedule_audit_runs.sql` (new)

```sql
ALTER TABLE schedule_audit
    ADD COLUMN trigger_type VARCHAR(16) NOT NULL DEFAULT 'scheduled' AFTER status,
    MODIFY COLUMN end_time DATETIME NULL;

CREATE INDEX idx_trigger_type ON schedule_audit (trigger_type);
CREATE INDEX idx_status       ON schedule_audit (status);
```

Backwards-compatible: existing rows keep their `end_time` and get
`trigger_type = 'scheduled'` by default.

#### 4.4.5 Model changes in `models/audit.go`

- `Audit.Status` may now be `running`.
- `Audit.Trigger string` field with JSON tag `trigger,omitempty`.
- `Audit.EndTime *time.Time` (pointer) so it can be nil for in-flight
  rows. Update `JSON` tag to `end_time,omitempty`.
- New methods needed in addition to existing `Save` / `Find*`:
  - `func (a Audit) UpdateResult(id int64, status, output, errStr string, endTime time.Time) (*Audit, error)`
    — issues a single `UPDATE schedule_audit SET status=?, output=?, error=?, end_time=? WHERE id=?`
    and returns the refreshed row. Used by the async runner to land the
    terminal state.
  - Existing `Save`, `FindById`, `FindByScheduleId`, `FindByDateRange`,
    `FindRecent`, `scanAudits` updated to read/write the new
    `trigger_type` column and the now-nullable `end_time`.

#### 4.4.6 Client changes in `clients/audit.go`

Add a thin wrapper for `UpdateResult`:

```go
// UpdateAuditResult lands the terminal state of an audit row that was
// previously inserted with status = running.
func UpdateAuditResult(id int64, status, output, errStr string, endTime time.Time) (*models.Audit, error) {
    var a models.Audit
    return a.UpdateResult(id, status, output, errStr, endTime)
}
```

### 4.5 Service refactor

In `services/scheduler.go`, replace the current closure-based execution
(`services/worker.go:44-70`) with three composable pieces:

```go
// StartRun creates the initial audit row in `running` state, records the
// start_time, and returns the persisted audit (with Id populated). It
// does NOT run the script. Used by both manual and scheduled paths.
func (s *SchedulerService) StartRun(sched *models.Schedule, trigger string) (*models.Audit, error)

// FinishRun executes the script synchronously and updates the existing
// audit row identified by audit.Id with the terminal status, output,
// error, and end_time. Used by the goroutine spawned for async runs and
// also by the worker (synchronously, since the worker is itself async
// to HTTP).
func (s *SchedulerService) FinishRun(sched *models.Schedule, audit *models.Audit) (*models.Audit, error)

// RunNow is the high-level entry point used by the HTTP handler. It
// loads the schedule, calls StartRun, spawns a goroutine that calls
// FinishRun, and returns the in-flight audit immediately. Errors here
// are pre-flight (schedule lookup, initial insert) and are surfaced to
// the caller; runtime script errors land in the audit row, not in the
// returned error.
func (s *SchedulerService) RunNow(id int64) (*models.Audit, error)
```

Concretely:

- `StartRun` calls `clients.CreateAudit` with
  `Status = AuditStatusRunning`, `StartTime = time.Now().UTC()`,
  `EndTime = nil`, `Trigger = trigger`, empty `Output` / `Error`.
- `FinishRun` calls `execCommand("/bin/sh", "-c", sched.ScriptPath).CombinedOutput()`,
  fills in success/failure plus `end_time`, and calls
  `clients.UpdateAuditResult`. Logging mirrors today's worker logs
  (`services/worker.go:62-65`).
- `RunNow` returns a typed `ErrScheduleNotFound` when the schedule lookup
  fails so the route can map it to `404`.

The worker's runner becomes:

```go
runner := func(sched *models.Schedule) {
    audit, err := Scheduler.StartRun(sched, models.AuditTriggerScheduled)
    if err != nil {
        log.Printf("[worker] schedule id=%d failed to start audit: %v", sched.Id, err)
        return
    }
    if _, err := Scheduler.FinishRun(sched, audit); err != nil {
        log.Printf("[worker] schedule id=%d failed to finalize audit: %v", sched.Id, err)
    }
}
```

The worker stays single-threaded per schedule (synchronous within its
goroutine), preserving today's behavior. Only the manual path spawns an
extra goroutine via `RunNow`.

### 4.6 Testability hooks

- Make the command runner a package-level variable in `services` so tests
  can stub it without invoking real shell commands:

  ```go
  var execCommand = exec.Command
  ```

  `FinishRun` uses `execCommand` instead of calling `exec.Command`
  directly.

- Make audit persistence package-level variables too:

  ```go
  var (
      saveAudit         = clients.CreateAudit
      updateAuditResult = clients.UpdateAuditResult
  )
  ```

  Lets `services/scheduler_test.go` assert calls without a DB and
  inspect the running → terminal transition.

- For the new routes, follow the function-variable pattern already used
  in `routes/audits.go:14-19`:

  ```go
  // routes/schedules.go
  var (
      runNowFn = func(id int64) (*models.Audit, error) { return services.Scheduler.RunNow(id) }
  )

  // routes/audits.go (alongside existing fn vars)
  var (
      getAuditFn = func(id int64) (*models.Audit, error) { return clients.GetAudit(id) }
  )
  ```

### 4.7 Concurrency

- In a single-process deployment, `RunNow` permits multiple in-flight
  runs of the same schedule; each gets its own row and `run_id`. This is
  intentional — the user explicitly asked to fire the schedule, and
  serializing manual runs would surprise them.
- The worker tick may also fire while a manual run of the same schedule
  is in flight; both audit rows simply coexist with different `trigger`
  values.
- Multi-process: not addressed in v1. Two `notes` instances pointed at
  the same DB *can* both produce running rows. Documented as a known
  limitation; a future `SELECT ... FOR UPDATE` or advisory-lock approach
  is non-disruptive to add.
- Process restart with running rows in DB: out of scope for v1. A
  running row whose owning process died will remain `running` forever.
  Future cleanup: a startup sweep that ages out `running` rows older
  than N hours into `failure` with a synthetic error message; not in v1.

## 5. File-by-File Changes

### 5.1 `migrations/V3__schedule_audit_runs.sql` (new)

Add `trigger_type` column with default `'scheduled'`, make `end_time`
nullable, and add indices on `trigger_type` and `status`. SQL in §4.4.4.

### 5.2 `models/audit.go`

- Constants: add `AuditStatusRunning`, `AuditTriggerScheduled`,
  `AuditTriggerManual` near `models/audit.go:11-14`.
- `Audit` struct (`models/audit.go:16-25`):
  - Add `Trigger string \`json:"trigger,omitempty"\``.
  - Change `EndTime time.Time` to `EndTime *time.Time` with
    `json:"end_time,omitempty"`.
- Update SQL strings, `Scan` destinations, and `Exec` argument lists in:
  - `Save` (`models/audit.go:27-54`)
  - `FindById` (`models/audit.go:56-82`)
  - `FindByScheduleId` (`models/audit.go:84-100`)
  - `FindByDateRange` (`models/audit.go:102-114`)
  - `FindRecent` (`models/audit.go:116-132`)
  - `scanAudits` (`models/audit.go:134-157`) — use `sql.NullTime` for
    `end_time`, dereference on assignment.
- Add new method `UpdateResult(id, status, output, errStr, endTime) (*Audit, error)`.

### 5.3 `clients/audit.go`

Add `UpdateAuditResult` (§4.4.6). Existing `CreateAudit` stays as-is; it
will be called with `Status = AuditStatusRunning` from `StartRun`.

### 5.4 `services/scheduler.go`

Add at the bottom (after `InvokeDueAt`, `services/scheduler.go:163-177`):

- Package-level vars `execCommand`, `saveAudit`, `updateAuditResult`
  (see §4.6).
- `var ErrScheduleNotFound = errors.New("schedule not found")` so the
  route layer can distinguish 404 from 500.
- `func (s *SchedulerService) StartRun(...)`.
- `func (s *SchedulerService) FinishRun(...)`.
- `func (s *SchedulerService) RunNow(id int64) (*models.Audit, error)`
  that wires `StartRun` + a goroutine running `FinishRun`.

### 5.5 `services/worker.go`

Replace lines `services/worker.go:44-70` with the thin runner shown in
§4.5. Keep the same log lines (just reorganized so we now log around
`StartRun` / `FinishRun` boundaries).

### 5.6 `routes/schedules.go`

- Add the `runNowFn` package-level variable (§4.6).
- Add `RunSchedule` handler:

  ```go
  // POST /schedules/:id/run
  func RunSchedule(w http.ResponseWriter, r *http.Request) {
      id, err := strconv.ParseInt(vestigo.Param(r, "id"), 10, 64)
      if err != nil {
          http.Error(w, "Invalid schedule id", http.StatusBadRequest)
          return
      }

      audit, err := runNowFn(id)
      if err != nil {
          if errors.Is(err, services.ErrScheduleNotFound) {
              http.Error(w, "Schedule not found", http.StatusNotFound)
              return
          }
          http.Error(w, err.Error(), http.StatusInternalServerError)
          return
      }

      js, _ := json.Marshal(audit)
      w.Header().Set(ContentType, JSON)
      w.Header().Set("Location", fmt.Sprintf("/runs/%d", audit.Id))
      w.WriteHeader(http.StatusAccepted)
      w.Write(js)
  }
  ```

### 5.7 `routes/audits.go`

- Add the `getAuditFn` package-level variable (§4.6).
- Add `GetRun` handler that returns a single `Audit` JSON by id; map
  `sql.ErrNoRows` (or equivalent) to 404.

  ```go
  // GET /runs/:id  (and aliased to GET /audits/:id)
  func GetRun(w http.ResponseWriter, r *http.Request) {
      id, err := strconv.ParseInt(vestigo.Param(r, "id"), 10, 64)
      if err != nil {
          http.Error(w, "Invalid run id", http.StatusBadRequest)
          return
      }

      audit, err := getAuditFn(id)
      if err != nil {
          if errors.Is(err, sql.ErrNoRows) {
              http.Error(w, "Run not found", http.StatusNotFound)
              return
          }
          http.Error(w, err.Error(), http.StatusInternalServerError)
          return
      }

      js, _ := json.Marshal(audit)
      w.Header().Set(ContentType, JSON)
      w.Write(js)
  }
  ```

### 5.8 `notes.go`

Register the new routes alongside the existing schedule / audit routes
(`notes.go:46-54`):

```go
router.Post("/schedules/:id/run", routes.RunSchedule)
router.Get("/runs/:id",           routes.GetRun)
router.Get("/audits/:id",         routes.GetRun) // alias; same handler
```

## 6. Tests

### 6.1 `services/scheduler_test.go`

Add tests using the `execCommand`, `saveAudit`, `updateAuditResult`
stubs. Each test installs stubs in `setUp` and restores in `defer`.

- `TestStartRun_PersistsRunningAudit`: stub `saveAudit` to capture its
  argument. Assert the captured audit has `Status == "running"`,
  `Trigger == "manual"`, `EndTime == nil`, and a populated `StartTime`.
- `TestFinishRun_Success_UpdatesToSuccess`: stub `execCommand` so its
  output is captured and exit code is 0. Assert `updateAuditResult`
  was called with `status = "success"`, the captured output, empty
  error, and a non-zero `end_time`.
- `TestFinishRun_Failure_UpdatesToFailure`: stub `execCommand` to fail.
  Assert `status = "failure"` and the error string is populated.
- `TestRunNow_ReturnsRunningAuditAndAsyncCompletes`: use a channel-based
  stub of `execCommand` that blocks until the test releases it. Call
  `RunNow`, assert the returned audit has `Status == "running"` and a
  populated `Id`. Release the stub, wait for the goroutine, then
  assert `updateAuditResult` was eventually called with a terminal
  status. (Use `sync.WaitGroup` or a buffered channel to synchronize.)
- `TestRunNow_ScheduleNotFound`: stub the schedule loader to return
  not-found; assert `RunNow` returns `ErrScheduleNotFound` and never
  inserts an audit.
- `TestRunNow_BypassesDisabledAndSnoozed`: schedule with
  `Status = disabled` and `SnoozedUntil` in the future. Assert that
  `RunNow` still inserts a `running` audit and the goroutine runs the
  script.
- `TestWorker_StillUsesScheduledTrigger`: invoke the worker runner
  with `StartRun`/`FinishRun` stubbed; assert the trigger argument was
  `"scheduled"`.

### 6.2 `routes/schedules_test.go` (new)

Mirror the testing style of `routes/audits_test.go`:

- Override `runNowFn` with in-memory stubs.
- `Test_RunSchedule_Accepted`: returns 202, body is the running audit
  JSON, `Location: /runs/<id>` header is set.
- `Test_RunSchedule_BadID`: non-numeric id → 400.
- `Test_RunSchedule_NotFound`: stub returns `services.ErrScheduleNotFound`
  → 404.
- `Test_RunSchedule_PersistError`: stub returns generic error → 500.

### 6.3 `routes/audits_test.go`

- Override `getAuditFn`.
- `Test_GetRun_Running`: stub returns an in-flight audit with
  `Status = "running"` and `EndTime = nil`. Assert 200 and that the
  JSON has `end_time` absent (or null) and `status: "running"`.
- `Test_GetRun_Terminal`: stub returns a `success` audit with
  populated `EndTime`. Assert 200 and full body.
- `Test_GetRun_NotFound`: stub returns `sql.ErrNoRows` → 404.
- `Test_GetRun_BadID`: non-numeric id → 400.
- Existing tests in this file continue to use their own stubs and
  should keep passing without modification.

### 6.4 Existing tests

- `services/scheduler_test.go` (`IsDueAt`, `InvokeDueAt`, snooze, etc.)
  must keep passing unchanged — these don't touch execution.
- `routes/audits_test.go` listing tests must keep passing; the model
  adds a new column and changes `EndTime` to a pointer. Confirm by
  running the suite. If JSON comparison tests rely on omitempty
  behavior of `end_time`, audit fixtures should be updated to set
  `EndTime` to a populated pointer.

## 7. Verification Steps

1. `make test` — all unit tests green, including the new ones in §6.
2. Apply migration `V3__schedule_audit_runs.sql` against a dev DB and
   confirm:
   - `DESCRIBE schedule_audit;` shows `trigger_type` (default
     `'scheduled'`) and `end_time` as nullable.
   - Existing rows keep their `end_time` and get
     `trigger_type = 'scheduled'`.
3. Smoke test against a running server with a *fast* script:
   - Create a schedule whose `script_path` is `echo hello-on-demand`.
   - `curl -i -X POST http://localhost:$PORT/schedules/<id>/run` →
     expect `202 Accepted`, `Location: /runs/<run_id>`, body with
     `status: "running"`, `trigger: "manual"`, no `end_time`.
   - `curl http://localhost:$PORT/runs/<run_id>` → eventually
     `status: "success"`, `output: "hello-on-demand\n"`, populated
     `end_time`.
4. Smoke test against a *slow* script (`script_path = sleep 5`):
   - `POST .../run` returns immediately (well under 5 s).
   - First `GET /runs/<run_id>` shows `status: "running"`.
   - After ~5 s, `GET /runs/<run_id>` shows `status: "success"`.
5. Toggle the schedule to `disabled`, run `POST .../run` again, and
   confirm it still executes (bypass behavior, §4.3).
6. Set `snoozed_until` to a future timestamp, run `POST .../run`, and
   confirm it still executes.
7. With a schedule whose `AllowedDays`/`AllowedTimes` match the next
   minute boundary, wait for the worker tick and confirm it writes one
   audit row that transitions `running -> success` and has
   `trigger: "scheduled"`. (This validates the worker now uses the
   same lifecycle.)
8. Trigger several manual runs of the same schedule in quick
   succession; confirm each gets a distinct `run_id` and they finish
   independently.

## 8. Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| In-flight `running` rows orphaned by process crash. | Documented limitation in v1. Future: startup sweep ages out stale `running` rows. |
| Caller polls forever because client lost the `run_id`. | `Location` header on the 202 response makes the canonical URL discoverable; `GET /schedules/:id/audits` shows recent runs (already exists, `routes/audits.go:23-44`). |
| Manual run during a scheduled tick double-executes. | Documented in §4.7; both runs are recorded with distinct `trigger` values and ids. Future: per-schedule mutex if it becomes painful. |
| Bypassing `disabled`/`snoozed` is footgun-y. | Default behavior is documented; future flag `?respect_status=true` is easy to add. |
| Schema change breaks existing deployments that share the DB. | `trigger_type` has a default value; `end_time` only loosens to nullable (existing NOT NULL rows are still valid). Old code that does not know about `running` will still see terminal rows correctly. |
| Tests that hit `exec.Command` accidentally run shell. | All execution tests must stub `execCommand`; CI never spawns a real subprocess for the runner path. |
| `EndTime` becoming a pointer breaks JSON consumers expecting a string. | `omitempty` keeps the field absent for in-flight rows; for terminal rows it serializes as before. Document the change. |

## 9. Rollout Order

Each step compiles and tests independently; no flag-flipping required.

1. Land migration + model changes (§5.1, §5.2). Existing
   `clients.CreateAudit` callers continue to work; new fields default
   sensibly.
2. Land `clients/audit.go` `UpdateAuditResult` helper (§5.3).
3. Land service refactor (§5.4, §5.5). Worker now uses
   `StartRun` + `FinishRun` and stamps `Trigger = "scheduled"`. No
   external behavior change for clients yet.
4. Land HTTP routes (§5.6, §5.7) and registration (§5.8). Async
   on-demand invocation goes live, polling endpoint goes live.
5. Land tests (§6) — ideally added together with each step above
   rather than at the end, but listed last for clarity.
