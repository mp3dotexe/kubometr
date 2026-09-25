ALTER TABLE users RENAME COLUMN telegram_id TO external_id;

ALTER TABLE users
    ADD COLUMN platform TEXT NOT NULL DEFAULT 'telegram'
    CHECK (platform IN ('telegram', 'max'));
ALTER TABLE users ALTER COLUMN platform DROP DEFAULT;

ALTER TABLE users DROP CONSTRAINT users_telegram_id_key;
ALTER TABLE users
    ADD CONSTRAINT users_platform_external_id_key UNIQUE (platform, external_id);
