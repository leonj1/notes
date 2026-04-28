-- V4__schedule_script_path_text.sql
--
-- Widen schedule.script_path from VARCHAR(512) to TEXT.
--
-- The LLM-driven path frequently stores curl invocations with embedded JSON
-- bodies that easily exceed 512 characters. With docker-compose configured
-- using --sql-mode=NO_ENGINE_SUBSTITUTION (i.e. NOT strict), MySQL silently
-- truncates oversize values, leaving the persisted script ending mid-quote.
-- The worker then runs `/bin/sh -c <truncated>` and dies with
-- "syntax error: unterminated quoted string", which manifests as the user's
-- "scheduler isn't working" reports even though the daemon is firing fine.
--
-- TEXT supports up to 65,535 bytes which is far more than any reasonable
-- script_path will ever need.

ALTER TABLE schedule MODIFY COLUMN script_path TEXT NOT NULL;
