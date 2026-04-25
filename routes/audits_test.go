package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"notes/models"
	"testing"
	"time"

	"github.com/husobee/vestigo"
)

// --- helpers ----------------------------------------------------------------

// setupRouter creates a vestigo router with the audit routes registered,
// matching the wiring in notes.go.
func setupRouter() *vestigo.Router {
	router := vestigo.NewRouter()
	router.Get("/schedules/:id/audits", ListAuditsBySchedule)
	router.Get("/audits/recent/:n", ListRecentAudits)
	router.Get("/audits", ListAuditsByDateRange)
	return router
}

// sampleAudits returns a small slice of audits useful for assertions.
func sampleAudits() []*models.Audit {
	return []*models.Audit{
		{
			Id:         1,
			ScheduleId: 10,
			ScriptPath: "/scripts/hello.sh",
			Status:     models.AuditStatusSuccess,
			Output:     "hello",
			StartTime:  time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
			EndTime:    time.Date(2026, 4, 25, 12, 0, 1, 0, time.UTC),
		},
		{
			Id:         2,
			ScheduleId: 10,
			ScriptPath: "/scripts/hello.sh",
			Status:     models.AuditStatusFailure,
			Output:     "",
			Error:      "exit status 1",
			StartTime:  time.Date(2026, 4, 25, 13, 0, 0, 0, time.UTC),
			EndTime:    time.Date(2026, 4, 25, 13, 0, 1, 0, time.UTC),
		},
	}
}

// mockListBySchedule replaces the real client call and captures the id.
func mockListBySchedule(capturedId *int64, result []*models.Audit) func() {
	orig := listAuditsByScheduleFn
	listAuditsByScheduleFn = func(id int64) ([]*models.Audit, error) {
		*capturedId = id
		return result, nil
	}
	return func() { listAuditsByScheduleFn = orig }
}

// mockListRecent replaces the real client call and captures n.
func mockListRecent(capturedN *int, result []*models.Audit) func() {
	orig := listRecentAuditsFn
	listRecentAuditsFn = func(n int) ([]*models.Audit, error) {
		*capturedN = n
		return result, nil
	}
	return func() { listRecentAuditsFn = orig }
}

// mockListByDateRange replaces the real client call and captures start/end.
func mockListByDateRange(capturedStart, capturedEnd *time.Time, result []*models.Audit) func() {
	orig := listAuditsByDateRangeFn
	listAuditsByDateRangeFn = func(start, end time.Time) ([]*models.Audit, error) {
		*capturedStart = start
		*capturedEnd = end
		return result, nil
	}
	return func() { listAuditsByDateRangeFn = orig }
}

// --- ListAuditsBySchedule tests --------------------------------------------

func TestListAuditsBySchedule_InvalidId(t *testing.T) {
	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/schedules/abc/audits", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestListAuditsBySchedule_ValidId(t *testing.T) {
	var capturedId int64
	cleanup := mockListBySchedule(&capturedId, sampleAudits())
	defer cleanup()

	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/schedules/10/audits", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if capturedId != 10 {
		t.Fatalf("expected schedule id 10, got %d", capturedId)
	}

	var audits []*models.Audit
	if err := json.Unmarshal(rr.Body.Bytes(), &audits); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(audits) != 2 {
		t.Fatalf("expected 2 audits, got %d", len(audits))
	}
	if audits[0].ScheduleId != 10 {
		t.Errorf("expected schedule_id 10, got %d", audits[0].ScheduleId)
	}
}

func TestListAuditsBySchedule_EmptyResult(t *testing.T) {
	var capturedId int64
	cleanup := mockListBySchedule(&capturedId, []*models.Audit{})
	defer cleanup()

	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/schedules/99/audits", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if rr.Body.String() != "[]" {
		t.Fatalf("expected empty array, got %s", rr.Body.String())
	}
}

// --- ListRecentAudits tests ------------------------------------------------

func TestListRecentAudits_InvalidN(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{"non-numeric", "/audits/recent/abc"},
		{"zero", "/audits/recent/0"},
		{"negative", "/audits/recent/-1"},
	}

	router := setupRouter()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d for %s, got %d", http.StatusBadRequest, tc.path, rr.Code)
			}
		})
	}
}

func TestListRecentAudits_ValidN(t *testing.T) {
	var capturedN int
	cleanup := mockListRecent(&capturedN, sampleAudits())
	defer cleanup()

	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/audits/recent/5", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if capturedN != 5 {
		t.Fatalf("expected n=5, got %d", capturedN)
	}

	var audits []*models.Audit
	if err := json.Unmarshal(rr.Body.Bytes(), &audits); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(audits) != 2 {
		t.Fatalf("expected 2 audits, got %d", len(audits))
	}
}

func TestListRecentAudits_ReturnsJSON(t *testing.T) {
	var capturedN int
	cleanup := mockListRecent(&capturedN, sampleAudits())
	defer cleanup()

	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/audits/recent/1", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if ct := rr.Header().Get("Content-Type"); ct != JSON {
		t.Fatalf("expected Content-Type %q, got %q", JSON, ct)
	}
}

// --- ListAuditsByDateRange tests -------------------------------------------

func TestListAuditsByDateRange_MissingParams(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{"both missing", "/audits"},
		{"start missing", "/audits?end=2026-04-25T23:59:59Z"},
		{"end missing", "/audits?start=2026-04-25T00:00:00Z"},
	}

	router := setupRouter()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d for %q, got %d", http.StatusBadRequest, tc.name, rr.Code)
			}
		})
	}
}

func TestListAuditsByDateRange_InvalidFormat(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{"bad start", "/audits?start=not-a-date&end=2026-04-25T23:59:59Z"},
		{"bad end", "/audits?start=2026-04-25T00:00:00Z&end=not-a-date"},
	}

	router := setupRouter()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d for %q, got %d", http.StatusBadRequest, tc.name, rr.Code)
			}
		})
	}
}

func TestListAuditsByDateRange_ValidParams(t *testing.T) {
	var capturedStart, capturedEnd time.Time
	cleanup := mockListByDateRange(&capturedStart, &capturedEnd, sampleAudits())
	defer cleanup()

	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet,
		"/audits?start=2026-04-25T00:00:00Z&end=2026-04-25T23:59:59Z", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	expectedStart := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2026, 4, 25, 23, 59, 59, 0, time.UTC)
	if !capturedStart.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, capturedStart)
	}
	if !capturedEnd.Equal(expectedEnd) {
		t.Fatalf("expected end %v, got %v", expectedEnd, capturedEnd)
	}

	var audits []*models.Audit
	if err := json.Unmarshal(rr.Body.Bytes(), &audits); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(audits) != 2 {
		t.Fatalf("expected 2 audits, got %d", len(audits))
	}
}

func TestListAuditsByDateRange_EmptyResult(t *testing.T) {
	var capturedStart, capturedEnd time.Time
	cleanup := mockListByDateRange(&capturedStart, &capturedEnd, []*models.Audit{})
	defer cleanup()

	router := setupRouter()
	req := httptest.NewRequest(http.MethodGet,
		"/audits?start=2020-01-01T00:00:00Z&end=2020-01-01T23:59:59Z", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if rr.Body.String() != "[]" {
		t.Fatalf("expected empty array, got %s", rr.Body.String())
	}
}

// --- JSON serialisation ----------------------------------------------------

func TestAuditJSON_RoundTrip(t *testing.T) {
	audits := sampleAudits()
	js, err := json.Marshal(audits)
	if err != nil {
		t.Fatalf("failed to marshal audits: %v", err)
	}

	var decoded []*models.Audit
	if err := json.Unmarshal(js, &decoded); err != nil {
		t.Fatalf("failed to unmarshal audits: %v", err)
	}

	if len(decoded) != len(audits) {
		t.Fatalf("expected %d audits, got %d", len(audits), len(decoded))
	}

	for i, a := range decoded {
		if a.Id != audits[i].Id {
			t.Errorf("[%d] Id = %d, want %d", i, a.Id, audits[i].Id)
		}
		if a.ScheduleId != audits[i].ScheduleId {
			t.Errorf("[%d] ScheduleId = %d, want %d", i, a.ScheduleId, audits[i].ScheduleId)
		}
		if a.Status != audits[i].Status {
			t.Errorf("[%d] Status = %q, want %q", i, a.Status, audits[i].Status)
		}
		if a.ScriptPath != audits[i].ScriptPath {
			t.Errorf("[%d] ScriptPath = %q, want %q", i, a.ScriptPath, audits[i].ScriptPath)
		}
		if a.Output != audits[i].Output {
			t.Errorf("[%d] Output = %q, want %q", i, a.Output, audits[i].Output)
		}
		if a.Error != audits[i].Error {
			t.Errorf("[%d] Error = %q, want %q", i, a.Error, audits[i].Error)
		}
	}
}
