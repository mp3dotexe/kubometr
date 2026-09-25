DELETE FROM users WHERE platform <> 'telegram';

ALTER TABLE users DROP CONSTRAINT users_platform_external_id_key;
ALTER TABLE users DROP COLUMN platform;
ALTER TABLE users RENAME COLUMN external_id TO telegram_id;
ALTER TABLE users ADD CONSTRAINT users_telegram_id_key UNIQUE (telegram_id);
