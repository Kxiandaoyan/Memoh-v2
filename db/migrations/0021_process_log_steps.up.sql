-- 0021_process_log_steps
-- Add new process_log_step enum values for comprehensive subsystem logging

ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'tool_call_started';
ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'tool_call_completed';
ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'memory_extract_started';
ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'memory_extract_completed';
ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'memory_extract_failed';
ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'token_trimmed';
ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'summary_loaded';
ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'summary_requested';
ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'skills_loaded';
ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'openviking_context';
ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'openviking_session';
ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'evolution_started';
ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'evolution_completed';
ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'evolution_failed';
