-- 0033_update_existing_bots_defaults
-- Backfill existing bots: set language to 'zh' and enable OpenViking

UPDATE bots SET language = 'zh' WHERE language = 'auto';
UPDATE bots SET enable_openviking = true WHERE enable_openviking = false;
