-- =============================================================================
-- Групповые чаты: сущности + участники
-- =============================================================================

CREATE TABLE IF NOT EXISTS chat_groups (
    id                  SERIAL          PRIMARY KEY,
    group_uuid          UUID            NOT NULL DEFAULT gen_random_uuid(),
    title               VARCHAR(255)    NOT NULL,
    avatar_file_uuid    UUID,
    created_by          INTEGER         NOT NULL,
    created_at          TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ(6),

    CONSTRAINT fk_chat_groups_creator
        FOREIGN KEY (created_by) REFERENCES user_relations (user_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_groups_group_uuid
    ON chat_groups (group_uuid)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_chat_groups_created_by ON chat_groups (created_by);
CREATE INDEX IF NOT EXISTS idx_chat_groups_avatar_file_uuid ON chat_groups (avatar_file_uuid);
CREATE INDEX IF NOT EXISTS idx_chat_groups_deleted_at ON chat_groups (deleted_at);

COMMENT ON TABLE chat_groups IS 'Групповой чат: название, аватар (file_uuid из files-сервиса), владелец';

-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS chat_group_members (
    id          SERIAL          PRIMARY KEY,
    group_id    INTEGER         NOT NULL,
    user_id     INTEGER         NOT NULL,
    role        VARCHAR(20)     NOT NULL DEFAULT 'member',
    created_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ(6),

    CONSTRAINT fk_chat_group_members_group
        FOREIGN KEY (group_id) REFERENCES chat_groups (id),
    CONSTRAINT fk_chat_group_members_user
        FOREIGN KEY (user_id) REFERENCES user_relations (user_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_group_members_group_user
    ON chat_group_members (group_id, user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_chat_group_members_group_id ON chat_group_members (group_id);
CREATE INDEX IF NOT EXISTS idx_chat_group_members_user_id ON chat_group_members (user_id);
CREATE INDEX IF NOT EXISTS idx_chat_group_members_deleted_at ON chat_group_members (deleted_at);

COMMENT ON TABLE chat_group_members IS 'Участники группового чата (owner|admin|member)';
