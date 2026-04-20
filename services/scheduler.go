package services

import (
	"notes/clients"
	"notes/models"
)

type SchedulerService struct{}

func NewSchedulerService() *SchedulerService {
	return &SchedulerService{}
}

func (s *SchedulerService) ListEnabled() ([]*models.Schedule, error) {
	all, err := clients.ListSchedules()
	if err != nil {
		return nil, err
	}

	enabled := make([]*models.Schedule, 0, len(all))
	for _, sched := range all {
		if sched.Status == models.ScheduleStatusEnabled {
			enabled = append(enabled, sched)
		}
	}
	return enabled, nil
}

var Scheduler = NewSchedulerService()
