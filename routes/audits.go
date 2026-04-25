package routes

import (
	"encoding/json"
	"github.com/husobee/vestigo"
	"net/http"
	"notes/clients"
	"notes/models"
	"strconv"
	"time"
)

// Common HTTP header constants used by all JSON-emitting handlers in this package.
const ContentType = "Content-Type"
const JSON = "application/json"

// Package-level function variables that delegate to the clients package.
// Tests can replace these to avoid hitting the database.
var (
	listAuditsByScheduleFn  = func(id int64) ([]*models.Audit, error) { return clients.ListAuditsBySchedule(id) }
	listRecentAuditsFn      = func(n int) ([]*models.Audit, error) { return clients.ListRecentAudits(n) }
	listAuditsByDateRangeFn = func(start, end time.Time) ([]*models.Audit, error) { return clients.ListAuditsByDateRange(start, end) }
)

// ListAuditsBySchedule returns all audit records for a given schedule id.
// GET /schedules/:id/audits
func ListAuditsBySchedule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(vestigo.Param(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid schedule id", http.StatusBadRequest)
		return
	}

	audits, err := listAuditsByScheduleFn(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(audits)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, JSON)
	w.Write(js)
}

// ListRecentAudits returns the most recent n audit records.
// GET /audits/recent/:n
func ListRecentAudits(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(vestigo.Param(r, "n"))
	if err != nil || n < 1 {
		http.Error(w, "Invalid count, must be a positive integer", http.StatusBadRequest)
		return
	}

	audits, err := listRecentAuditsFn(n)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(audits)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, JSON)
	w.Write(js)
}

// ListAuditsByDateRange returns audit records whose start_time falls within
// the given range. Both query parameters use RFC 3339 format.
// GET /audits?start=2024-01-01T00:00:00Z&end=2024-01-31T23:59:59Z
func ListAuditsByDateRange(w http.ResponseWriter, r *http.Request) {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		http.Error(w, "Both 'start' and 'end' query parameters are required (RFC 3339 format)", http.StatusBadRequest)
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		http.Error(w, "Invalid 'start' parameter: "+err.Error(), http.StatusBadRequest)
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		http.Error(w, "Invalid 'end' parameter: "+err.Error(), http.StatusBadRequest)
		return
	}

	audits, err := listAuditsByDateRangeFn(start, end)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(audits)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, JSON)
	w.Write(js)
}
