-- V3__schedule_description.sql
-- Adds an optional human-readable description column to schedules so users
-- can record the purpose of each schedule alongside its cron expression.

ALTER TABLE schedule
    ADD COLUMN description VARCHAR(1024) NOT NULL DEFAULT '' AFTER script_path;
