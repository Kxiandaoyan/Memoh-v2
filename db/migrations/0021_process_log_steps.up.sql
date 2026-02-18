-- 0021_process_log_steps
-- Add new process_log_step enum values for comprehensive subsystem logging

DO $$ BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'tool_call_started';
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'tool_call_completed';
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'memory_extract_started';
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'memory_extract_completed';
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'memory_extract_failed';
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'token_trimmed';
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'summary_loaded';
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'summary_requested';
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'skills_loaded';
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'openviking_context';
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'openviking_session';
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'evolution_started';
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'evolution_completed';
EXCEPTION WHEN duplicate_object THEN null; END $$;
DO $$ BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'evolution_failed';
EXCEPTION WHEN duplicate_object THEN null; END $$;
