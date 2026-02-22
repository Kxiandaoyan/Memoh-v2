-- 0032_bot_default_language_openviking (rollback)

ALTER TABLE bots
  ALTER COLUMN language SET DEFAULT 'auto',
  ALTER COLUMN enable_openviking SET DEFAULT false;
