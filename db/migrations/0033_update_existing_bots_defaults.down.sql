-- 0033_update_existing_bots_defaults (rollback)
UPDATE bots SET enable_openviking = false;
