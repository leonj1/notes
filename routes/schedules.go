package routes

import (
	"encoding/json"
	"errors"
	"github.com/husobee/vestigo"
	"net/http"
	"notes/models"
	"notes/services"
	"strconv"
)

func CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var schedule models.Schedule
	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	saved, err := services.Scheduler.Add(schedule)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(saved)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, JSON)
	w.WriteHeader(http.StatusCreated)
	w.Write(js)
}

func GetSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(vestigo.Param(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid schedule id", http.StatusBadRequest)
		return
	}

	schedule, err := services.Scheduler.Get(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(schedule)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, JSON)
	w.Write(js)
}

func ListSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := services.Scheduler.ListEnabled()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(schedules)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, JSON)
	w.Write(js)
}

func DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(vestigo.Param(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid schedule id", http.StatusBadRequest)
		return
	}

	if err := services.Scheduler.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, JSON)
	w.Write([]byte(`{"status":"deleted"}`))
}

// runScheduleFn is the package-level seam for tests so the route can be
// exercised without the service-layer DB dependency.
var runScheduleFn = func(id int64) (*models.Audit, error) {
	return services.Scheduler.RunNow(id)
}

// RunSchedule executes the schedule with the given id immediately and
// returns the resulting audit row. The schedule's cron / allowed_days /
// allowed_times / snooze fields are ignored — this is a manual override
// intended for "Run now" UI actions and ad-hoc verification.
//
// POST /schedules/:id/run
//
//	200 OK    -> Audit JSON (status="success" or "failure")
//	400 BAD   -> id is not a valid integer
//	404 NF    -> no schedule with that id
//	500 ISE   -> unexpected error (e.g. DB unreachable, audit save failure)
//
// A "failure" audit is still returned with HTTP 200 so the caller can read
// the captured stderr/stdout. Distinguish "the script failed" from "we
// couldn't run it" by inspecting the returned audit's status field.
func RunSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(vestigo.Param(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid schedule id", http.StatusBadRequest)
		return
	}

	audit, err := runScheduleFn(id)
	if err != nil {
		if errors.Is(err, services.ErrScheduleNotFound) {
			http.Error(w, "schedule not found", http.StatusNotFound)
			return
		}
		// Audit may still be populated when persistence failed mid-run.
		// Surface that as 500 with the err message so the UI shows what
		// actually went wrong instead of a blanket "internal error".
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(audit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(ContentType, JSON)
	w.Write(js)
}
