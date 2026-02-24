ALTER TABLE bots DROP CONSTRAINT bots_status_check;
ALTER TABLE bots ADD CONSTRAINT bots_status_check CHECK (status IN ('creating', 'ready', 'deleting'));
