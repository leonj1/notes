package clients

import (
	"notes/models"
	"time"
)

// CreateAudit inserts a new audit record for a scheduled task execution
// and returns the persisted record (with Id populated).
func CreateAudit(audit models.Audit) (*models.Audit, error) {
	audit.Id = 0
	return audit.Save()
}

// GetAudit returns a single audit record by id.
func GetAudit(id int64) (*models.Audit, error) {
	var a models.Audit
	return a.FindById(id)
}

// ListAuditsBySchedule returns all audit records for a specific schedule,
// ordered by most recent first.
func ListAuditsBySchedule(scheduleId int64) ([]*models.Audit, error) {
	var a models.Audit
	return a.FindByScheduleId(scheduleId)
}

// ListAuditsByDateRange returns all audit records whose start_time falls
// within the inclusive [start, end] range, ordered by most recent first.
func ListAuditsByDateRange(start, end time.Time) ([]*models.Audit, error) {
	var a models.Audit
	return a.FindByDateRange(start, end)
}

// ListRecentAudits returns the most recent n audit records, ordered by
// most recent first.
func ListRecentAudits(n int) ([]*models.Audit, error) {
	var a models.Audit
	return a.FindRecent(n)
}
