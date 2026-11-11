-- sender_id на связи message → group (автор группового сообщения)
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS sender_id INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_groups_sender_id ON groups (sender_id);

COMMENT ON COLUMN groups.sender_id IS 'messages-local user_id автора сообщения в группе';
