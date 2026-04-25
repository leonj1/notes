-- Adds bi-weekly recurrence support and snooze state to the schedule table.
--
-- - interval_weeks: cadence in weeks. NULL/0/1 mean "every week".
--   N>1 means "every Nth week relative to anchor_date".
-- - anchor_date:    reference date used to compute week parity.
--                   NULL means treat IntervalWeeks as 1 (every week).
-- - snoozed_until:  optional ad-hoc suppression timestamp. While
--                   set and in the future, the schedule is silenced.
ALTER TABLE schedule
    ADD COLUMN interval_weeks INT       NOT NULL DEFAULT 1,
    ADD COLUMN anchor_date    DATETIME  NULL,
    ADD COLUMN snoozed_until  DATETIME  NULL;
