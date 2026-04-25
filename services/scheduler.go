package services

import (
	"notes/clients"
	"notes/models"
)

type SchedulerService struct{}

func NewSchedulerService() *SchedulerService {
	return &SchedulerService{}
}

func (s *SchedulerService) Add(schedule models.Schedule) (*models.Schedule, error) {
	return clients.CreateSchedule(schedule)
}

func (s *SchedulerService) Get(id int64) (*models.Schedule, error) {
	return clients.GetSchedule(id)
}

func (s *SchedulerService) Update(schedule models.Schedule) (*models.Schedule, error) {
	return clients.UpdateSchedule(schedule)
}

func (s *SchedulerService) Delete(id int64) error {
	return clients.DeleteSchedule(id)
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
