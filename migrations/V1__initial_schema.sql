-- V1__initial_schema.sql
-- Creates the complete schema required by the notes service.
--
-- Tables:
--   notes     stored notes (with optional expiration)
--   tags      key/value tags attached to notes
--   schedule  reminder schedules with bi-weekly recurrence and snooze

CREATE TABLE IF NOT EXISTS notes (
    id              BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    note            TEXT         NOT NULL,
    creator         VARCHAR(255) NOT NULL,
    create_date     DATETIME     NOT NULL,
    expiration_date DATETIME     NOT NULL DEFAULT '0000-00-00 00:00:00'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tags (
    id          BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    note_id     BIGINT       NOT NULL,
    `key`       VARCHAR(255) NOT NULL,
    `value`     VARCHAR(255) NOT NULL,
    creator     VARCHAR(255) NOT NULL,
    create_date DATETIME     NOT NULL,
    INDEX idx_note_id   (note_id),
    INDEX idx_key_value (`key`, `value`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS schedule (
    id             BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    cron_schedule  VARCHAR(255) NOT NULL,
    allowed_days   VARCHAR(255) NOT NULL DEFAULT '',
    allowed_times  VARCHAR(255) NOT NULL DEFAULT '',
    silence_days   VARCHAR(255) NOT NULL DEFAULT '',
    silence_times  VARCHAR(255) NOT NULL DEFAULT '',
    script_path    VARCHAR(512) NOT NULL,
    status         VARCHAR(20)  NOT NULL DEFAULT 'disabled',
    create_date    DATETIME     NOT NULL,
    -- recurrence cadence (0 or 1 = every week; N>1 = every Nth week)
    interval_weeks INT          NOT NULL DEFAULT 1,
    -- reference date for week-parity calculation (NULL = ignore interval)
    anchor_date    DATETIME     NULL,
    -- ad-hoc suppression (NULL = not snoozed)
    snoozed_until  DATETIME     NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
