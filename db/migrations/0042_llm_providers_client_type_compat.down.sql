-- 0042_llm_providers_client_type_compat (rollback)
-- Remove client_type compatibility column and constraint.

ALTER TABLE llm_providers
  DROP CONSTRAINT IF EXISTS llm_providers_client_type_check;

ALTER TABLE llm_providers
  DROP COLUMN IF EXISTS client_type;
