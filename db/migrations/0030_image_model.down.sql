-- 0030_image_model (rollback)
ALTER TABLE bots DROP COLUMN IF EXISTS image_model_id;
