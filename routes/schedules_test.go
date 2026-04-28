package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"notes/models"
	"notes/services"
	"testing"
	"time"

	"github.com/husobee/vestigo"
)

// helper that wires only the schedule-run route and stubs the runScheduleFn
// seam so tests never reach the DB or the shell.
func setupRunRouter(stub func(int64) (*models.Audit, error)) (*vestigo.Router, func()) {
	router := vestigo.NewRouter()
	router.Post("/schedules/:id/run", RunSchedule)

	orig := runScheduleFn
	runScheduleFn = stub
	return router, func() { runScheduleFn = orig }
}

func TestRunSchedule_Success(t *testing.T) {
	wantAudit := &models.Audit{
		Id:         77,
		ScheduleId: 12,
		ScriptPath: "echo hi",
		Status:     models.AuditStatusSuccess,
		Output:     "hi\n",
		StartTime:  time.Date(2026, 4, 27, 22, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 4, 27, 22, 0, 1, 0, time.UTC),
	}
	var capturedId int64
	router, cleanup := setupRunRouter(func(id int64) (*models.Audit, error) {
		capturedId = id
		return wantAudit, nil
	})
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/schedules/12/run", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if capturedId != 12 {
		t.Errorf("runScheduleFn got id=%d, want 12", capturedId)
	}
	if ct := rr.Header().Get("Content-Type"); ct != JSON {
		t.Errorf("Content-Type = %q, want %q", ct, JSON)
	}

	var got models.Audit
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if got.Id != wantAudit.Id || got.ScheduleId != wantAudit.ScheduleId {
		t.Errorf("body = %+v, want id=%d schedule_id=%d", got, wantAudit.Id, wantAudit.ScheduleId)
	}
	if got.Status != models.AuditStatusSuccess {
		t.Errorf("body.Status = %q, want %q", got.Status, models.AuditStatusSuccess)
	}
}

// A failing script (status=failure) is still a successful HTTP response —
// the audit IS the response body. The UI distinguishes by the status field.
func TestRunSchedule_FailureAuditReturnsHTTP200(t *testing.T) {
	wantAudit := &models.Audit{
		Id:         78,
		ScheduleId: 12,
		ScriptPath: "false",
		Status:     models.AuditStatusFailure,
		Error:      "exit status 1",
	}
	router, cleanup := setupRunRouter(func(id int64) (*models.Audit, error) {
		return wantAudit, nil
	})
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/schedules/12/run", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 (script failure is still a successful invocation), got %d", rr.Code)
	}
	var got models.Audit
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if got.Status != models.AuditStatusFailure {
		t.Errorf("body.Status = %q, want %q", got.Status, models.AuditStatusFailure)
	}
	if got.Error == "" {
		t.Errorf("body.Error should be populated on failure")
	}
}

func TestRunSchedule_InvalidId(t *testing.T) {
	router, cleanup := setupRunRouter(func(id int64) (*models.Audit, error) {
		t.Fatalf("runScheduleFn should not be called for invalid id")
		return nil, nil
	})
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/schedules/not-a-number/run", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestRunSchedule_NotFound(t *testing.T) {
	router, cleanup := setupRunRouter(func(id int64) (*models.Audit, error) {
		return nil, services.ErrScheduleNotFound
	})
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/schedules/9999/run", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing schedule, got %d", rr.Code)
	}
}

func TestRunSchedule_InternalError(t *testing.T) {
	router, cleanup := setupRunRouter(func(id int64) (*models.Audit, error) {
		return nil, errors.New("DB on fire")
	})
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/schedules/12/run", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on unexpected error, got %d", rr.Code)
	}
}
