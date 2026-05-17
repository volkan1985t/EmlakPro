-- v3: Task management, per-user telegram_chat_id, bot sessions

ALTER TABLE users ADD COLUMN IF NOT EXISTS telegram_chat_id VARCHAR(50);
ALTER TABLE users ADD COLUMN IF NOT EXISTS notify_telegram BOOLEAN NOT NULL DEFAULT true;

CREATE TABLE IF NOT EXISTS bot_sessions (
    chat_id    BIGINT PRIMARY KEY,
    user_id    BIGINT REFERENCES users(id),
    step       VARCHAR(50),
    data       JSONB,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tasks (
    id              BIGSERIAL PRIMARY KEY,
    parent_id       BIGINT REFERENCES tasks(id) ON DELETE CASCADE,
    title           VARCHAR(500) NOT NULL,
    description     TEXT,
    status          VARCHAR(30) NOT NULL DEFAULT 'bekliyor',
    priority        VARCHAR(20) NOT NULL DEFAULT 'normal',
    due_date        DATE,
    created_by      BIGINT NOT NULL REFERENCES users(id),
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
);
-- status: bekliyor | devam_ediyor | tamamlandi | iptal
-- priority: dusuk | normal | yuksek | acil

CREATE TABLE IF NOT EXISTS task_assignees (
    id         BIGSERIAL PRIMARY KEY,
    task_id    BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(task_id, user_id)
);

CREATE TABLE IF NOT EXISTS task_comments (
    id         BIGSERIAL PRIMARY KEY,
    task_id    BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id),
    body       TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS task_images (
    id         BIGSERIAL PRIMARY KEY,
    task_id    BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    path       VARCHAR(500) NOT NULL,
    uploaded_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tasks_parent   ON tasks(parent_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status   ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_task_assignees ON task_assignees(task_id);
CREATE INDEX IF NOT EXISTS idx_task_comments  ON task_comments(task_id);
CREATE INDEX IF NOT EXISTS idx_task_images    ON task_images(task_id);
