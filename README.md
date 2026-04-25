# notes

A small Go HTTP service for storing notes with tags, plus a built-in
schedule engine for time-based reminders (with bi-weekly recurrence and
snooze support). Backed by MySQL, packaged with Docker.

```
POST /notes              -> create a note
GET  /notes              -> list all notes
GET  /activenotes        -> list non-expired notes
PUT  /notes/{id}         -> attach a tag to a note
DELETE /notes/{id}       -> delete a note (and its tags)
GET  /tags/{key}/{value} -> filter notes by tag
```

## What you get

- **Notes API** — JSON over HTTP, expirable, taggable.
- **Tag filtering** — `key=value` lookups with one URL.
- **Schedule engine** — schedules with `AllowedDays`, `AllowedTimes`,
  bi-weekly cadence (`IntervalWeeks` + `AnchorDate`), and ad-hoc
  `SnoozedUntil` suppression. Single matcher (`services.IsDueAt`)
  returns whether a schedule should fire at a given `time.Time`.
- **Containerized build & test** — `Dockerfile.build` produces a static
  binary; `make test` runs the suite in `Dockerfile.test`. No local Go
  toolchain required.

## Quick start (3 minutes)

You need only Docker and a MySQL instance reachable from your host.

```sh
# 1. Run the tests (no DB required — pure unit tests in a container)
make test

# 2. Build the server image
docker build -f Dockerfile.build -t notes .

# 3. Apply the schedule schema migration to your MySQL DB
mysql -u <user> -p <db> < migrations/001_add_recurrence_and_snooze.sql

# 4. Run the server (point it at your MySQL)
docker run --rm -p 8080:8080 notes \
    -user=<user> -pass=<pass> -db=<db> -port=8080
```

The service exits if the DB is unreachable, so verify your MySQL
credentials first. The `notes`, `tags`, and `schedule` tables must
exist; `migrations/` contains incremental DDL for the schedule pieces.

## Examples

### Create a note

```sh
curl -X POST http://localhost:8080/notes \
    -H 'Content-Type: application/json' \
    -d '{
        "Note":           "buy milk",
        "Creator":        "alice",
        "ExpirationDate": "2026-12-31T00:00:00Z"
    }'
```

### Tag it

```sh
curl -X PUT http://localhost:8080/notes/1 \
    -H 'Content-Type: application/json' \
    -d '{"Key":"category","Value":"groceries"}'
```

### List active notes

```sh
curl http://localhost:8080/activenotes
```

### Filter by tag

```sh
curl http://localhost:8080/tags/category/groceries
```

### Working with schedules (Go)

The schedule engine is exposed as `services.Scheduler`. Schedules persist
through MySQL via the `clients` package; the matcher is pure and
testable.

```go
import (
    "time"
    "notes/models"
    "notes/services"
)

// A medicine reminder for the whole weekend, single row.
weekend, _ := services.Scheduler.Add(models.Schedule{
    AllowedDays:  "Fri,Sat,Sun",
    AllowedTimes: "08:00",
    ScriptPath:   "/scripts/remind.sh",
    Status:       models.ScheduleStatusEnabled,
})

// Every other Friday at 5pm, anchored to a known Friday.
biweekly, _ := services.Scheduler.Add(models.Schedule{
    AllowedDays:   "Fri",
    AllowedTimes:  "17:00",
    IntervalWeeks: 2,
    AnchorDate:    time.Date(2024, 4, 26, 0, 0, 0, 0, time.UTC),
    ScriptPath:    "/scripts/payroll.sh",
    Status:        models.ScheduleStatusEnabled,
})

// Ask "what fires right now?" against a list of schedules.
all, _ := services.Scheduler.ListEnabled()
services.Scheduler.InvokeDueAt(all, time.Now(), func(s *models.Schedule) {
    // dispatch s.ScriptPath however your runtime prefers
})

// Snooze the weekend reminder until tomorrow morning.
services.Scheduler.Snooze(weekend.Id, time.Now().Add(24*time.Hour))
```

### Schedule field cheat sheet

| Field           | Type         | Meaning                                                                              |
| --------------- | ------------ | ------------------------------------------------------------------------------------ |
| `AllowedDays`   | `string`     | Comma-separated three-letter weekday names (`"Mon,Wed,Fri"`). Empty = any day.       |
| `AllowedTimes`  | `string`     | Comma-separated `HH:MM` times. Empty = any time.                                     |
| `IntervalWeeks` | `int`        | `0` or `1` = every week. `N>1` = every Nth week relative to `AnchorDate`.            |
| `AnchorDate`    | `time.Time`  | Reference date for `IntervalWeeks`. Times before the anchor never match.             |
| `SnoozedUntil`  | `*time.Time` | If set and in the future, the schedule is suppressed. `nil` means "not snoozed".     |
| `Status`        | `string`     | `"enabled"` or `"disabled"`. Disabled schedules are skipped by `InvokeDueAt`.        |

## Run the tests

```sh
make test
```

This builds `Dockerfile.test` and runs `go test -v ./...` inside the
container — no local Go install needed. The matcher tests are
table-driven with BDD-style `given/when/then` names.

## Project layout

```
notes.go                       # main: flag parsing, DB init, HTTP routes
routes/                        # one HTTP handler per file
services/                      # business logic (notes, scheduler)
clients/                       # thin wrappers over models for service use
models/                        # SQL persistence (Note, Tag, Schedule)
migrations/                    # incremental DDL
Dockerfile.build               # builds the static notes binary
Dockerfile.test                # runs the test suite
Makefile                       # `make test`
```

## License

MIT — see `LICENSE`.
