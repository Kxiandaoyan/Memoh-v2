-- Migration: process_logs
-- Creates process_log table for tracking complete conversation flow from user input to LLM response

-- Create enum type for log levels if not exists
DO $$ BEGIN
    CREATE TYPE process_log_level AS ENUM ('debug', 'info', 'warn', 'error');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Create enum type for log steps if not exists
DO $$ BEGIN
    CREATE TYPE process_log_step AS ENUM (
        'user_message_received',
        'history_loaded',
        'memory_searched',
        'memory_loaded',
        'prompt_built',
        'llm_request_sent',
        'llm_response_received',
        'tool_call_started',
        'tool_call_completed',
        'response_sent',
        'memory_stored',
        'memory_extract_started',
        'memory_extract_completed',
        'memory_extract_failed',
        'stream_started',
        'stream_completed',
        'stream_error',
        'token_trimmed',
        'summary_loaded',
        'summary_requested',
        'skills_loaded',
        'openviking_context',
        'openviking_session',
        'evolution_started',
        'evolution_completed',
        'evolution_failed'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Create process_logs table
CREATE TABLE IF NOT EXISTS process_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL,
    chat_id UUID NOT NULL,
    trace_id UUID NOT NULL DEFAULT gen_random_uuid(),
    user_id VARCHAR(255),
    channel VARCHAR(100),
    step process_log_step NOT NULL,
    level process_log_level DEFAULT 'info',
    message TEXT,
    data JSONB DEFAULT '{}',
    duration_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_process_logs_bot_id ON process_logs(bot_id);
CREATE INDEX IF NOT EXISTS idx_process_logs_chat_id ON process_logs(chat_id);
CREATE INDEX IF NOT EXISTS idx_process_logs_trace_id ON process_logs(trace_id);
CREATE INDEX IF NOT EXISTS idx_process_logs_created_at ON process_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_process_logs_step ON process_logs(step);

-- Create composite index for recent logs query
CREATE INDEX IF NOT EXISTS idx_process_logs_recent ON process_logs(bot_id, created_at DESC);

-- Function to cleanup old logs (keep last 500 per bot per day - optional)
CREATE OR REPLACE FUNCTION cleanup_old_process_logs(retention_days INTEGER DEFAULT 7)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM process_logs
    WHERE created_at < NOW() - INTERVAL '1 day' * retention_days
    AND id NOT IN (
        SELECT id FROM process_logs
        WHERE created_at >= NOW() - INTERVAL '1 day' * retention_days
        ORDER BY created_at DESC
        LIMIT 500
    );
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
