-- =============================================================================
-- chickchirick-messages: полная схема БД
-- PostgreSQL. Соответствует GORM-моделям internal/model/message/*
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- -----------------------------------------------------------------------------
-- 1. user_relations  (локальные id ↔ UUID из user-сервиса)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_relations (
    user_id     SERIAL          PRIMARY KEY,
    user_uuid   UUID            NOT NULL,
    created_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ(6)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_relations_user_uuid
    ON user_relations (user_uuid)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_relations_deleted_at
    ON user_relations (deleted_at);

COMMENT ON TABLE user_relations IS 'Связь messages-local user_id с UUID пользователя (user-сервис)';

-- -----------------------------------------------------------------------------
-- 2. messages
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS messages (
    id          SERIAL          PRIMARY KEY,
    text        TEXT            NOT NULL,
    created_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ(6)
);

CREATE INDEX IF NOT EXISTS idx_messages_deleted_at
    ON messages (deleted_at);

COMMENT ON TABLE messages IS 'Тело сообщения (plaintext или ciphertext E2EE)';

-- -----------------------------------------------------------------------------
-- 3. personals  (1:1 привязка sender/recipient к message)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS personals (
    id              SERIAL          PRIMARY KEY,
    message_id      INTEGER         NOT NULL,
    sender_id       INTEGER         NOT NULL,
    recipient_id    INTEGER         NOT NULL,
    created_at      TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ(6),

    CONSTRAINT fk_personals_message
        FOREIGN KEY (message_id) REFERENCES messages (id),
    CONSTRAINT fk_personals_sender
        FOREIGN KEY (sender_id) REFERENCES user_relations (user_id),
    CONSTRAINT fk_personals_recipient
        FOREIGN KEY (recipient_id) REFERENCES user_relations (user_id)
);

CREATE INDEX IF NOT EXISTS idx_personals_message_id ON personals (message_id);
CREATE INDEX IF NOT EXISTS idx_personals_sender_id ON personals (sender_id);
CREATE INDEX IF NOT EXISTS idx_personals_recipient_id ON personals (recipient_id);
CREATE INDEX IF NOT EXISTS idx_personals_deleted_at ON personals (deleted_at);

-- -----------------------------------------------------------------------------
-- 4. files
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS files (
    id          SERIAL          PRIMARY KEY,
    message_id  INTEGER         NOT NULL,
    file_uuid   UUID            NOT NULL,
    created_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ(6),

    CONSTRAINT fk_files_message
        FOREIGN KEY (message_id) REFERENCES messages (id)
);

CREATE INDEX IF NOT EXISTS idx_files_message_id ON files (message_id);
CREATE INDEX IF NOT EXISTS idx_files_file_uuid ON files (file_uuid);
CREATE INDEX IF NOT EXISTS idx_files_deleted_at ON files (deleted_at);

-- -----------------------------------------------------------------------------
-- 5. groups
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS groups (
    id          SERIAL          PRIMARY KEY,
    message_id  INTEGER         NOT NULL,
    group_id    UUID            NOT NULL,
    created_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ(6),

    CONSTRAINT fk_groups_message
        FOREIGN KEY (message_id) REFERENCES messages (id)
);

CREATE INDEX IF NOT EXISTS idx_groups_message_id ON groups (message_id);
CREATE INDEX IF NOT EXISTS idx_groups_group_id ON groups (group_id);
CREATE INDEX IF NOT EXISTS idx_groups_deleted_at ON groups (deleted_at);

-- -----------------------------------------------------------------------------
-- 6. deleteds  (пометки удаления сообщения пользователем)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS deleteds (
    id          SERIAL          PRIMARY KEY,
    message_id  INTEGER         NOT NULL,
    user_id     INTEGER         NOT NULL,
    created_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ(6),

    CONSTRAINT fk_deleteds_message
        FOREIGN KEY (message_id) REFERENCES messages (id),
    CONSTRAINT fk_deleteds_user
        FOREIGN KEY (user_id) REFERENCES user_relations (user_id)
);

CREATE INDEX IF NOT EXISTS idx_deleteds_message_id ON deleteds (message_id);
CREATE INDEX IF NOT EXISTS idx_deleteds_user_id ON deleteds (user_id);
CREATE INDEX IF NOT EXISTS idx_deleteds_deleted_at ON deleteds (deleted_at);

-- -----------------------------------------------------------------------------
-- 7. statuses  (прочтение)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS statuses (
    id              SERIAL          PRIMARY KEY,
    message_id      INTEGER         NOT NULL,
    read_user_id    INTEGER,
    created_at      TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ(6),

    CONSTRAINT fk_statuses_message
        FOREIGN KEY (message_id) REFERENCES messages (id),
    CONSTRAINT fk_statuses_read_user
        FOREIGN KEY (read_user_id) REFERENCES user_relations (user_id)
);

CREATE INDEX IF NOT EXISTS idx_statuses_message_id ON statuses (message_id);
CREATE INDEX IF NOT EXISTS idx_statuses_read_user_id ON statuses (read_user_id);
CREATE INDEX IF NOT EXISTS idx_statuses_deleted_at ON statuses (deleted_at);

-- -----------------------------------------------------------------------------
-- 8. message_meta
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS message_meta (
    id                   SERIAL          PRIMARY KEY,
    message_uuid         UUID            NOT NULL DEFAULT gen_random_uuid(),
    message_id           INTEGER         NOT NULL,
    message_status_id    INTEGER         NOT NULL DEFAULT 0,
    respond_message_id   INTEGER,
    created_at           TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ(6),

    CONSTRAINT fk_message_meta_message
        FOREIGN KEY (message_id) REFERENCES messages (id),
    CONSTRAINT fk_message_meta_respond
        FOREIGN KEY (respond_message_id) REFERENCES messages (id)
);

CREATE INDEX IF NOT EXISTS idx_message_meta_message_id ON message_meta (message_id);
CREATE INDEX IF NOT EXISTS idx_message_meta_message_uuid ON message_meta (message_uuid);
CREATE INDEX IF NOT EXISTS idx_message_meta_deleted_at ON message_meta (deleted_at);

-- -----------------------------------------------------------------------------
-- 9. settings
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS settings (
    id                  SERIAL          PRIMARY KEY,
    rule_installer_id   INTEGER         NOT NULL,
    rule_user_id        INTEGER         NOT NULL,
    rule                JSONB           NOT NULL DEFAULT '[]'::jsonb,
    created_at          TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ(6),

    CONSTRAINT fk_settings_installer
        FOREIGN KEY (rule_installer_id) REFERENCES user_relations (user_id),
    CONSTRAINT fk_settings_user
        FOREIGN KEY (rule_user_id) REFERENCES user_relations (user_id)
);

CREATE INDEX IF NOT EXISTS idx_settings_rule_installer_id ON settings (rule_installer_id);
CREATE INDEX IF NOT EXISTS idx_settings_rule_user_id ON settings (rule_user_id);
CREATE INDEX IF NOT EXISTS idx_settings_deleted_at ON settings (deleted_at);

-- -----------------------------------------------------------------------------
-- 10. e2ee_identities
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS e2ee_identities (
    user_id             INTEGER         PRIMARY KEY,
    identity_key        TEXT            NOT NULL,
    signed_pre_key      TEXT            NOT NULL,
    signed_pre_key_id   INTEGER         NOT NULL DEFAULT 1,
    signed_pre_key_sig  TEXT            NOT NULL,
    updated_at          TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_e2ee_identities_user
        FOREIGN KEY (user_id) REFERENCES user_relations (user_id) ON DELETE CASCADE
);

COMMENT ON TABLE e2ee_identities IS 'E2EE public identity + signed prekey (Signal-style)';

-- -----------------------------------------------------------------------------
-- 11. e2ee_one_time_prekeys
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS e2ee_one_time_prekeys (
    id          SERIAL          PRIMARY KEY,
    user_id     INTEGER         NOT NULL,
    key_id      INTEGER         NOT NULL,
    public_key  TEXT            NOT NULL,
    created_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_e2ee_opk_user
        FOREIGN KEY (user_id) REFERENCES user_relations (user_id) ON DELETE CASCADE,
    CONSTRAINT uq_e2ee_opk_user_key
        UNIQUE (user_id, key_id)
);

CREATE INDEX IF NOT EXISTS idx_e2ee_opk_user_id ON e2ee_one_time_prekeys (user_id);

COMMENT ON TABLE e2ee_one_time_prekeys IS 'X3DH one-time prekeys; deleted after single use';
