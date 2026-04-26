package sdk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// ListAuditsBySchedule returns the audit history for one schedule, ordered
// by start_time DESC.
func (c *Client) ListAuditsBySchedule(ctx context.Context, scheduleID int64) ([]Audit, error) {
	if scheduleID < 1 {
		return nil, errors.New("notes sdk: schedule id must be positive")
	}
	out := make([]Audit, 0)
	path := fmt.Sprintf("/schedules/%d/audits", scheduleID)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListRecentAudits returns the n most recent audits across all schedules.
// n must be a positive integer.
func (c *Client) ListRecentAudits(ctx context.Context, n int) ([]Audit, error) {
	if n < 1 {
		return nil, errors.New("notes sdk: n must be positive")
	}
	out := make([]Audit, 0)
	path := fmt.Sprintf("/audits/recent/%d", n)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListAuditsByDateRange returns audits whose start_time falls within
// [start, end]. Both bounds are inclusive and serialized as RFC 3339 in UTC.
func (c *Client) ListAuditsByDateRange(ctx context.Context, start, end time.Time) ([]Audit, error) {
	if start.IsZero() || end.IsZero() {
		return nil, errors.New("notes sdk: start and end are required")
	}
	if end.Before(start) {
		return nil, errors.New("notes sdk: end must be on or after start")
	}
	q := url.Values{}
	q.Set("start", start.UTC().Format(time.RFC3339))
	q.Set("end", end.UTC().Format(time.RFC3339))

	out := make([]Audit, 0)
	if err := c.do(ctx, http.MethodGet, "/audits", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
