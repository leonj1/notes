package sdk

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// recordedRequest captures the parts of an http.Request the tests want to
// assert against. We snapshot eagerly because the server hands the request
// back into the handler and reads the body.
type recordedRequest struct {
	Method string
	Path   string
	Query  string
	Body   string
	Header http.Header
}

// newTestServer wires up an httptest server whose handler is provided by the
// test, and returns a Client pointed at it plus the recorded request.
func newTestServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, rec *recordedRequest)) (*Client, *recordedRequest, func()) {
	t.Helper()
	rec := &recordedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.Method = r.Method
		rec.Path = r.URL.Path
		rec.Query = r.URL.RawQuery
		rec.Header = r.Header.Clone()
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			rec.Body = string(b)
		}
		handler(w, r, rec)
	}))
	client := NewClient(srv.URL)
	return client, rec, srv.Close
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	c := NewClient("http://example.com/")
	if c.baseURL != "http://example.com" {
		t.Fatalf("expected trailing slash trimmed, got %q", c.baseURL)
	}
}

func TestCreateSchedule_SendsDescriptionAndDecodesResponse(t *testing.T) {
	const wantDescription = "Weekly Monday reminder"

	client, rec, cleanup := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ *recordedRequest) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{
			"id": "42",
			"cron_schedule": "0 9 * * 1",
			"script_path": "/scripts/hello.sh",
			"description": "`+wantDescription+`",
			"status": "enabled",
			"create_date": "2026-04-25T12:00:00Z"
		}`)
	})
	defer cleanup()

	in := Schedule{
		CronSchedule: "0 9 * * 1",
		ScriptPath:   "/scripts/hello.sh",
		Description:  wantDescription,
		Status:       ScheduleStatusEnabled,
	}
	out, err := client.CreateSchedule(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateSchedule returned error: %v", err)
	}

	if rec.Method != http.MethodPost {
		t.Errorf("expected POST, got %s", rec.Method)
	}
	if rec.Path != "/schedules" {
		t.Errorf("expected path /schedules, got %s", rec.Path)
	}
	if got := rec.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", got)
	}
	if !strings.Contains(rec.Header.Get("User-Agent"), "notes-go-sdk/") {
		t.Errorf("expected SDK User-Agent, got %q", rec.Header.Get("User-Agent"))
	}

	var sent map[string]any
	if err := json.Unmarshal([]byte(rec.Body), &sent); err != nil {
		t.Fatalf("could not decode request body: %v", err)
	}
	if sent["description"] != wantDescription {
		t.Errorf("expected request description %q, got %v", wantDescription, sent["description"])
	}

	if out.Id != 42 {
		t.Errorf("expected Id 42, got %d", out.Id)
	}
	if out.Description != wantDescription {
		t.Errorf("expected response Description %q, got %q", wantDescription, out.Description)
	}
}

func TestGetSchedule_SuccessAndValidation(t *testing.T) {
	client, rec, cleanup := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ *recordedRequest) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"7","cron_schedule":"0 9 * * 1","description":"hi"}`)
	})
	defer cleanup()

	got, err := client.GetSchedule(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetSchedule returned error: %v", err)
	}
	if rec.Method != http.MethodGet || rec.Path != "/schedules/7" {
		t.Errorf("expected GET /schedules/7, got %s %s", rec.Method, rec.Path)
	}
	if got.Id != 7 || got.Description != "hi" {
		t.Errorf("unexpected schedule: %+v", got)
	}

	if _, err := client.GetSchedule(context.Background(), 0); err == nil {
		t.Errorf("expected error for non-positive id")
	}
}

func TestListSchedules_EmptyAndPopulated(t *testing.T) {
	t.Run("populated", func(t *testing.T) {
		client, _, cleanup := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ *recordedRequest) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"id":"1","status":"enabled"},{"id":"2","status":"enabled"}]`)
		})
		defer cleanup()

		got, err := client.ListSchedules(context.Background())
		if err != nil {
			t.Fatalf("ListSchedules returned error: %v", err)
		}
		if len(got) != 2 || got[0].Id != 1 || got[1].Id != 2 {
			t.Errorf("unexpected schedules: %+v", got)
		}
	})

	t.Run("empty", func(t *testing.T) {
		client, _, cleanup := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ *recordedRequest) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[]`)
		})
		defer cleanup()

		got, err := client.ListSchedules(context.Background())
		if err != nil {
			t.Fatalf("ListSchedules returned error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %+v", got)
		}
	})
}

func TestDeleteSchedule(t *testing.T) {
	client, rec, cleanup := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ *recordedRequest) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"deleted"}`)
	})
	defer cleanup()

	if err := client.DeleteSchedule(context.Background(), 5); err != nil {
		t.Fatalf("DeleteSchedule returned error: %v", err)
	}
	if rec.Method != http.MethodDelete || rec.Path != "/schedules/5" {
		t.Errorf("expected DELETE /schedules/5, got %s %s", rec.Method, rec.Path)
	}

	if err := client.DeleteSchedule(context.Background(), 0); err == nil {
		t.Errorf("expected error for non-positive id")
	}
}

func TestAPIError_PropagatesServerBody(t *testing.T) {
	client, _, cleanup := newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ *recordedRequest) {
		http.Error(w, "Invalid schedule id", http.StatusBadRequest)
	})
	defer cleanup()

	_, err := client.CreateSchedule(context.Background(), Schedule{
		CronSchedule: "x",
		ScriptPath:   "y",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if !apiErr.IsBadRequest() {
		t.Errorf("expected IsBadRequest, got status %d", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Error(), "Invalid schedule id") {
		t.Errorf("expected body in error string, got %q", apiErr.Error())
	}
}
