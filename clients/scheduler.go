package clients

import (
	"notes/models"
)

// CreateSchedule inserts a new schedule and returns the persisted record
// (with Id populated).
func CreateSchedule(schedule models.Schedule) (*models.Schedule, error) {
	// Force the insert path regardless of what the caller supplied.
	schedule.Id = 0
	return schedule.Save()
}

// GetSchedule returns a single schedule by id.
func GetSchedule(id int64) (*models.Schedule, error) {
	var s models.Schedule
	return s.FindById(id)
}

// ListSchedules returns all schedule rows.
func ListSchedules() ([]*models.Schedule, error) {
	var s models.Schedule
	return s.All()
}

// UpdateSchedule updates the schedule identified by schedule.Id using the
// provided field values and returns the persisted record.
func UpdateSchedule(schedule models.Schedule) (*models.Schedule, error) {
	return schedule.Save()
}

// DeleteSchedule removes the schedule row with the given id.
func DeleteSchedule(id int64) error {
	var s models.Schedule
	return s.DeleteById(id)
}
