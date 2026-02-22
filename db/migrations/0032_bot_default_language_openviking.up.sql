-- 0032_bot_default_language_openviking
-- Change default language from 'auto' to 'zh' and enable_openviking default from false to true

ALTER TABLE bots
  ALTER COLUMN language SET DEFAULT 'zh',
  ALTER COLUMN enable_openviking SET DEFAULT true;
