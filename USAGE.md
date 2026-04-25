# Notes API Usage

This document describes every HTTP endpoint exposed by the `notes` service so an
AI tool (or any HTTP client) can call them without reading the source. All
endpoints return `application/json` and respond with plain-text errors carrying
non-2xx status codes.

## Base URLs

| Environment | URL | Notes |
| --- | --- | --- |
| Backend (direct) | `http://localhost:${NOTES_PORT:-8080}` | Routes defined in `server.go:11-37`. |
| UI proxy | `http://localhost:${UI_PORT:-8081}` | nginx serves the static UI and proxies `/(schedules\|audits\|notes\|activenotes\|tags)` to the backend (`nginx.ui.conf:9-17`). Use this URL when calling from a browser context. |

The same paths work on either base URL.

## Conventions

- **Content type**: requests with bodies must send `Content-Type: application/json`.
- **Timestamps**: query parameters use RFC 3339 (e.g. `2024-01-31T23:59:59Z`). Response timestamps follow Go's default `time.Time` JSON encoding (RFC 3339 with nanoseconds).
- **Errors**: failures return a non-2xx status with a plain-text body (e.g. `Invalid schedule id`). The UI proxy can additionally return HTML 404s from nginx for paths that are neither static files nor proxied.
- **JSON casing**:
  - `Schedule` and `Audit` use `snake_case` (see `models/schedule.go:17-36`, `models/audit.go:16-25`).
  - `Note` has no struct tags so it is serialized in PascalCase (`Id`, `Note`, `Creator`, `CreateDate`, `ExpirationDate`, `Tags`) — see `models/note.go:11-18`.
  - `Tag` uses lowercase keys (`id`, `note_id`, `creator`, `key`, `value`) — see `models/tag.go:11-18`.

---

## Notes

### `GET /notes` — list all notes

Returns every note with embedded tags.

```bash
curl -s http://localhost:8080/notes
```

```json
[
  {
    "Id": 1,
    "Note": "buy milk",
    "Creator": "alice",
    "CreateDate": "2024-05-01T10:00:00Z",
    "ExpirationDate": "0001-01-01T00:00:00Z",
    "Tags": [
      { "id": "10", "note_id": "1", "creator": "alice", "key": "category", "value": "groceries" }
    ]
  }
]
```

> Implementation note: `models/note.go:23-24` currently passes a parameter to a
> parameter-less `SELECT`, so this endpoint may respond with
> `sql: expected 0 arguments, got 1` until fixed.

### `POST /notes` — create a note

Request body:

```json
{
  "Note": "pick up dry cleaning",
  "Creator": "alice",
  "ExpirationDate": "2024-12-31T00:00:00Z"
}
```

Example:

```bash
curl -s -X POST http://localhost:8080/notes \
  -H 'Content-Type: application/json' \
  -d '{"Note":"pick up dry cleaning","Creator":"alice","ExpirationDate":"2024-12-31T00:00:00Z"}'
```

Response (200): the created note with its assigned `Id` and server `CreateDate`.

### `PUT /notes/:id` — attach a tag to a note

Adds a single tag (key/value pair) to the note. Returns 400 if the same key/value already exists for the note (`services.ErrTagAlreadyExists`).

```bash
curl -s -X PUT http://localhost:8080/notes/1 \
  -H 'Content-Type: application/json' \
  -d '{"key":"priority","value":"high","creator":"alice"}'
```

Response (200):

```json
{ "id": "10", "note_id": "1", "creator": "alice", "key": "priority", "value": "high" }
```

### `DELETE /notes/:id` — delete a note and its tags

```bash
curl -s -X DELETE http://localhost:8080/notes/1
```

Response (200):

```json
"{\"status\":\"deleted\"}"
```

### `GET /activenotes` — list non-expired notes

Returns notes whose `expiration_date` is in the future (or zero/unset).

```bash
curl -s http://localhost:8080/activenotes
```

Response shape: same as `GET /notes`.

### `GET /tags/:key/:value` — filter notes by tag

```bash
curl -s http://localhost:8080/tags/priority/high
```

Response shape: array of notes (same as `GET /notes`) whose tags include the given key/value.

---

## Schedules

A `Schedule` describes a cron-driven script execution. Defined in `models/schedule.go:17-36`.

### Schedule object

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string-encoded int | Server-assigned. |
| `cron_schedule` | string | Required on create. |
| `script_path` | string | Required on create. |
| `status` | `"enabled"` \| `"disabled"` | Defaults to `"disabled"` if omitted. |
| `allowed_days`, `allowed_times`, `silence_days`, `silence_times` | string | Free-form windows. |
| `interval_weeks` | int | `0` or `1` = every week; `N` = every Nth week relative to `anchor_date`. |
| `anchor_date` | RFC 3339 | Required when `interval_weeks > 1`. |
| `snoozed_until` | RFC 3339 \| `null` | Suppresses firings before this time. |
| `create_date` | RFC 3339 | Server-assigned on create. |

### `POST /schedules` — create a schedule

```bash
curl -s -X POST http://localhost:8080/schedules \
  -H 'Content-Type: application/json' \
  -d '{
    "cron_schedule": "*/5 * * * *",
    "script_path": "/scripts/health.sh",
    "status": "enabled",
    "allowed_days": "mon-fri",
    "allowed_times": "09:00-17:00"
  }'
```

Response (201):

```json
{
  "id": "1",
  "cron_schedule": "*/5 * * * *",
  "script_path": "/scripts/health.sh",
  "status": "enabled",
  "allowed_days": "mon-fri",
  "allowed_times": "09:00-17:00",
  "create_date": "2024-05-01T10:00:00Z"
}
```

Errors: 400 if `cron_schedule` or `script_path` is missing, or `status` is not `enabled`/`disabled`.

### `GET /schedules` — list enabled schedules

```bash
curl -s http://localhost:8080/schedules
```

Returns an array of schedules with `status == "enabled"`.

### `GET /schedules/:id` — fetch a single schedule

```bash
curl -s http://localhost:8080/schedules/1
```

Response: a single schedule object. 400 if `id` is not a valid integer.

### `DELETE /schedules/:id` — delete a schedule

```bash
curl -s -X DELETE http://localhost:8080/schedules/1
```

Response (200):

```json
{"status":"deleted"}
```

---

## Audits

An `Audit` records a single scheduled-script execution. Defined in `models/audit.go:16-25`.

### Audit object

| Field | Type | Notes |
| --- | --- | --- |
| `id` | string-encoded int | |
| `schedule_id` | string-encoded int | FK to `Schedule.id`. |
| `script_path` | string | |
| `status` | `"success"` \| `"failure"` | |
| `output` | string | stdout from the run. |
| `error` | string | stderr / failure message. |
| `start_time`, `end_time` | RFC 3339 | |

### `GET /schedules/:id/audits` — audits for one schedule

Ordered by `start_time DESC`.

```bash
curl -s http://localhost:8080/schedules/1/audits
```

```json
[
  {
    "id": "42",
    "schedule_id": "1",
    "script_path": "/scripts/health.sh",
    "status": "success",
    "output": "ok\n",
    "start_time": "2024-05-01T10:00:00Z",
    "end_time":   "2024-05-01T10:00:01Z"
  }
]
```

### `GET /audits/recent/:n` — most recent N audits

`n` must be a positive integer (400 otherwise).

```bash
curl -s http://localhost:8080/audits/recent/25
```

Response: array of audits, newest first.

### `GET /audits?start=...&end=...` — audits in a date range

Both `start` and `end` are required and must be RFC 3339. Filter is inclusive on both ends and applied to `start_time`.

```bash
curl -s 'http://localhost:8080/audits?start=2024-01-01T00:00:00Z&end=2024-01-31T23:59:59Z'
```

Response: array of audits ordered by `start_time DESC`. 400 if either parameter is missing or unparseable.

---

## Quick reference

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/notes` | List all notes (with tags). |
| POST | `/notes` | Create a note. |
| PUT | `/notes/:id` | Attach a tag to a note. |
| DELETE | `/notes/:id` | Delete a note and its tags. |
| GET | `/activenotes` | List non-expired notes. |
| GET | `/tags/:key/:value` | List notes with a given tag. |
| POST | `/schedules` | Create a schedule. |
| GET | `/schedules` | List enabled schedules. |
| GET | `/schedules/:id` | Fetch one schedule. |
| DELETE | `/schedules/:id` | Delete a schedule. |
| GET | `/schedules/:id/audits` | Audits for one schedule. |
| GET | `/audits/recent/:n` | Most recent N audits. |
| GET | `/audits?start=&end=` | Audits within a date range. |

Routing is registered in `server.go:14-34`.
