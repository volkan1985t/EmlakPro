-- Faz 2: cihaz token'ları (saha cihazları için salt-okunur sorgu erişimi)
-- Her token bir danışmana (user_id) bağlıdır; sadece /api/sorgu/* uçlarında geçerlidir.

CREATE TABLE IF NOT EXISTS device_tokens (
    id           BIGSERIAL PRIMARY KEY,
    token        TEXT        NOT NULL UNIQUE,
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label        TEXT        NOT NULL DEFAULT '',
    is_active    BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_device_tokens_token ON device_tokens(token);
CREATE INDEX IF NOT EXISTS idx_device_tokens_user  ON device_tokens(user_id);
