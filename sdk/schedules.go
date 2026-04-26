package sdk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// CreateSchedule creates a new schedule. The server requires CronSchedule and
// ScriptPath to be non-empty; Status defaults to "disabled" if omitted. The
// returned schedule reflects the server-assigned Id and CreateDate.
func (c *Client) CreateSchedule(ctx context.Context, s Schedule) (*Schedule, error) {
	var out Schedule
	if err := c.do(ctx, http.MethodPost, "/schedules", nil, s, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetSchedule fetches a single schedule by its numeric id.
func (c *Client) GetSchedule(ctx context.Context, id int64) (*Schedule, error) {
	if id < 1 {
		return nil, errors.New("notes sdk: schedule id must be positive")
	}
	var out Schedule
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/schedules/%d", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListSchedules returns every enabled schedule. Disabled schedules are not
// included in the response (see services.Scheduler.ListEnabled on the
// server).
func (c *Client) ListSchedules(ctx context.Context) ([]Schedule, error) {
	out := make([]Schedule, 0)
	if err := c.do(ctx, http.MethodGet, "/schedules", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteSchedule removes the schedule with the given id. The server replies
// with {"status":"deleted"} on success, which the SDK consumes silently.
func (c *Client) DeleteSchedule(ctx context.Context, id int64) error {
	if id < 1 {
		return errors.New("notes sdk: schedule id must be positive")
	}
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/schedules/%d", id), nil, nil, nil)
}
