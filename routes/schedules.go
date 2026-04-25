package routes

import (
	"encoding/json"
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
