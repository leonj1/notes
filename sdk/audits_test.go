package sdk

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestListAuditsBySchedule(t *testing.T) {
	client, rec, cleanup := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ *recordedRequest) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[
			{"id":"1","schedule_id":"10","status":"success","start_time":"2026-04-25T12:00:00Z","end_time":"2026-04-25T12:00:01Z"},
			{"id":"2","schedule_id":"10","status":"failure","error":"boom","start_time":"2026-04-25T13:00:00Z","end_time":"2026-04-25T13:00:01Z"}
		]`)
	})
	defer cleanup()

	got, err := client.ListAuditsBySchedule(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListAuditsBySchedule returned error: %v", err)
	}
	if rec.Method != http.MethodGet || rec.Path != "/schedules/10/audits" {
		t.Errorf("expected GET /schedules/10/audits, got %s %s", rec.Method, rec.Path)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 audits, got %d", len(got))
	}
	if got[0].ScheduleId != 10 {
		t.Errorf("expected schedule_id 10, got %d", got[0].ScheduleId)
	}
	if got[1].Status != AuditStatusFailure {
		t.Errorf("expected failure status, got %q", got[1].Status)
	}

	if _, err := client.ListAuditsBySchedule(context.Background(), 0); err == nil {
		t.Errorf("expected error for non-positive id")
	}
}

func TestListRecentAudits(t *testing.T) {
	client, rec, cleanup := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ *recordedRequest) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"id":"99","schedule_id":"1","status":"success"}]`)
	})
	defer cleanup()

	got, err := client.ListRecentAudits(context.Background(), 25)
	if err != nil {
		t.Fatalf("ListRecentAudits returned error: %v", err)
	}
	if rec.Path != "/audits/recent/25" {
		t.Errorf("expected path /audits/recent/25, got %s", rec.Path)
	}
	if len(got) != 1 || got[0].Id != 99 {
		t.Errorf("unexpected response: %+v", got)
	}

	if _, err := client.ListRecentAudits(context.Background(), 0); err == nil {
		t.Errorf("expected error for non-positive n")
	}
}

func TestListAuditsByDateRange(t *testing.T) {
	client, rec, cleanup := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ *recordedRequest) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[]`)
	})
	defer cleanup()

	start := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 25, 23, 59, 59, 0, time.UTC)
	got, err := client.ListAuditsByDateRange(context.Background(), start, end)
	if err != nil {
		t.Fatalf("ListAuditsByDateRange returned error: %v", err)
	}
	if rec.Path != "/audits" {
		t.Errorf("expected path /audits, got %s", rec.Path)
	}
	wantQuery := "end=2026-04-25T23%3A59%3A59Z&start=2026-04-25T00%3A00%3A00Z"
	if rec.Query != wantQuery {
		t.Errorf("expected query %q, got %q", wantQuery, rec.Query)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %+v", got)
	}

	cases := []struct {
		name       string
		start, end time.Time
	}{
		{"missing start", time.Time{}, end},
		{"missing end", start, time.Time{}},
		{"end before start", end, start},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := client.ListAuditsByDateRange(context.Background(), tc.start, tc.end); err == nil {
				t.Errorf("expected error")
			}
		})
	}
}
