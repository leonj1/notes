# Notes API Usage

This document describes every HTTP endpoint exposed by the `notes` service so an
AI tool (or any HTTP client) can call them without reading the source. All
endpoints return `application/json` and respond with plain-text errors carrying
non-2xx status codes.

## Base URLs

| Environment | URL | Notes |
| --- | --- | --- |
| Backend (direct) | `http://localhost:${NOTES_PORT:-8080}` | Routes defined in `server.go:11-26`. |
| UI proxy | `http://localhost:${UI_PORT:-8081}` | nginx serves the static UI and proxies `/(schedules\|audits)` to the backend (`nginx.ui.conf:8-16`). Use this URL when calling from a browser context. |

The same paths work on either base URL.

## Conventions

- **Content type**: requests with bodies must send `Content-Type: application/json`.
- **Timestamps**: query parameters use RFC 3339 (e.g. `2024-01-31T23:59:59Z`). Response timestamps follow Go's default `time.Time` JSON encoding (RFC 3339 with nanoseconds).
- **Errors**: failures return a non-2xx status with a plain-text body (e.g. `Invalid schedule id`). The UI proxy can additionally return HTML 404s from nginx for paths that are neither static files nor proxied.
- **JSON casing**: `Schedule` and `Audit` use `snake_case` (see `models/schedule.go:17-36`, `models/audit.go:16-25`).

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
| POST | `/schedules` | Create a schedule. |
| GET | `/schedules` | List enabled schedules. |
| GET | `/schedules/:id` | Fetch one schedule. |
| DELETE | `/schedules/:id` | Delete a schedule. |
| GET | `/schedules/:id/audits` | Audits for one schedule. |
| GET | `/audits/recent/:n` | Most recent N audits. |
| GET | `/audits?start=&end=` | Audits within a date range. |

Routing is registered in `server.go:14-23`.
