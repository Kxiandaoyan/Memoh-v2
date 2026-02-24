-- Add 'failed' to bots status constraint so container setup failures
-- can be properly recorded instead of silently violating the check.
ALTER TABLE bots DROP CONSTRAINT bots_status_check;
ALTER TABLE bots ADD CONSTRAINT bots_status_check CHECK (status IN ('creating', 'ready', 'failed', 'deleting'));

-- Fix any bots stuck in 'creating' for more than 10 minutes (already failed).
UPDATE bots SET status = 'failed', updated_at = NOW()
WHERE status = 'creating'
  AND created_at < NOW() - INTERVAL '10 minutes';
