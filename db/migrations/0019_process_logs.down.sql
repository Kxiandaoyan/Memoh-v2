-- Migration down: process_logs

DROP TABLE IF EXISTS process_logs CASCADE;
DROP FUNCTION IF EXISTS cleanup_old_process_logs(INTEGER);
DROP TYPE IF EXISTS process_log_level CASCADE;
DROP TYPE IF EXISTS process_log_step CASCADE;
