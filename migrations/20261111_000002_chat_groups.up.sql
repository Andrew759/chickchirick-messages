-- =============================================================================
-- Групповые чаты: сущности + участники
-- =============================================================================

CREATE TABLE IF NOT EXISTS group_meta (
    id                  SERIAL          PRIMARY KEY,
    group_uuid          UUID            NOT NULL DEFAULT gen_random_uuid(),
    title               VARCHAR(255)    NOT NULL,
    avatar_file_uuid    UUID,
    created_by          INTEGER         NOT NULL,
    created_at          TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ(6),

    CONSTRAINT fk_group_meta_creator
        FOREIGN KEY (created_by) REFERENCES user_relations (user_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_group_meta_group_uuid
    ON group_meta (group_uuid)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_group_meta_created_by ON group_meta (created_by);
CREATE INDEX IF NOT EXISTS idx_group_meta_avatar_file_uuid ON group_meta (avatar_file_uuid);
CREATE INDEX IF NOT EXISTS idx_group_meta_deleted_at ON group_meta (deleted_at);

COMMENT ON TABLE group_meta IS 'Групповой чат: название, аватар (file_uuid из files-сервиса), владелец';

-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS group_members (
    id          SERIAL          PRIMARY KEY,
    group_id    INTEGER         NOT NULL,
    user_id     INTEGER         NOT NULL,
    role        VARCHAR(20)     NOT NULL DEFAULT 'member',
    created_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ(6),

    CONSTRAINT fk_group_members_group
        FOREIGN KEY (group_id) REFERENCES group_meta (id),
    CONSTRAINT fk_group_members_user
        FOREIGN KEY (user_id) REFERENCES user_relations (user_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_group_members_group_user
    ON group_members (group_id, user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_group_members_group_id ON chat_group_members (group_id);
CREATE INDEX IF NOT EXISTS idx_group_members_user_id ON chat_group_members (user_id);
CREATE INDEX IF NOT EXISTS idx_group_members_deleted_at ON chat_group_members (deleted_at);

COMMENT ON TABLE group_members IS 'Участники группового чата (owner|admin|member)';
