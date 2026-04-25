-- V2__schedule_audit.sql
-- Audit table that records every execution of a scheduled task.

CREATE TABLE IF NOT EXISTS schedule_audit (
    id          BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    schedule_id BIGINT       NOT NULL,
    script_path VARCHAR(512) NOT NULL,
    status      VARCHAR(20)  NOT NULL,
    output      TEXT         NOT NULL DEFAULT '',
    error       TEXT         NOT NULL DEFAULT '',
    start_time  DATETIME     NOT NULL,
    end_time    DATETIME     NOT NULL,
    INDEX idx_schedule_id (schedule_id),
    INDEX idx_start_time  (start_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
