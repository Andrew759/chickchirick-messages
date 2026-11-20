-- =============================================================================
-- Каналы (broadcast) + участники + связь сообщений
-- E2EE: ciphertext в messages.text; ротация ключа через channel_members_changed
-- =============================================================================

CREATE TABLE IF NOT EXISTS channels (
    id                  SERIAL          PRIMARY KEY,
    channel_uuid        UUID            NOT NULL DEFAULT gen_random_uuid(),
    title               VARCHAR(255)    NOT NULL,
    description         TEXT            NOT NULL DEFAULT '',
    avatar_file_uuid    UUID,
    is_public           BOOLEAN         NOT NULL DEFAULT FALSE,
    e2ee_enabled        BOOLEAN         NOT NULL DEFAULT TRUE,
    created_by          INTEGER         NOT NULL,
    created_at          TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ(6),

    CONSTRAINT fk_channels_creator
        FOREIGN KEY (created_by) REFERENCES user_relations (user_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_channel_uuid
    ON channels (channel_uuid)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_channels_created_by ON channels (created_by);
CREATE INDEX IF NOT EXISTS idx_channels_is_public ON channels (is_public) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_channels_avatar_file_uuid ON channels (avatar_file_uuid);
CREATE INDEX IF NOT EXISTS idx_channels_deleted_at ON channels (deleted_at);

COMMENT ON TABLE channels IS 'Канал: broadcast-лента; e2ee_enabled=true — клиент шифрует text';
COMMENT ON COLUMN channels.e2ee_enabled IS 'Клиент обязан шифровать payload; сервер не видит plaintext';

-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS channel_members (
    id          SERIAL          PRIMARY KEY,
    channel_id  INTEGER         NOT NULL,
    user_id     INTEGER         NOT NULL,
    role        VARCHAR(20)     NOT NULL DEFAULT 'subscriber',
    created_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ(6),

    CONSTRAINT fk_channel_members_channel
        FOREIGN KEY (channel_id) REFERENCES channels (id),
    CONSTRAINT fk_channel_members_user
        FOREIGN KEY (user_id) REFERENCES user_relations (user_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_channel_members_channel_user
    ON channel_members (channel_id, user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_channel_members_channel_id ON channel_members (channel_id);
CREATE INDEX IF NOT EXISTS idx_channel_members_user_id ON channel_members (user_id);
CREATE INDEX IF NOT EXISTS idx_channel_members_deleted_at ON channel_members (deleted_at);

COMMENT ON TABLE channel_members IS 'Участники канала (owner|admin|subscriber); писать могут только owner/admin';

-- -----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS channel_messages (
    id              SERIAL          PRIMARY KEY,
    message_id      INTEGER         NOT NULL,
    channel_id      UUID            NOT NULL,
    sender_id       INTEGER         NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ(6),

    CONSTRAINT fk_channel_messages_message
        FOREIGN KEY (message_id) REFERENCES messages (id)
);

CREATE INDEX IF NOT EXISTS idx_channel_messages_message_id ON channel_messages (message_id);
CREATE INDEX IF NOT EXISTS idx_channel_messages_channel_id ON channel_messages (channel_id);
CREATE INDEX IF NOT EXISTS idx_channel_messages_sender_id ON channel_messages (sender_id);
CREATE INDEX IF NOT EXISTS idx_channel_messages_deleted_at ON channel_messages (deleted_at);

COMMENT ON TABLE channel_messages IS 'Связь message → channel (аналог groups для групповых чатов)';
COMMENT ON COLUMN channel_messages.sender_id IS 'messages-local user_id автора поста в канале';
