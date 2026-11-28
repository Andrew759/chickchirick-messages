CREATE TABLE IF NOT EXISTS message_pins (
    id               SERIAL          PRIMARY KEY,
    message_id       INTEGER         NOT NULL,
    peer_user_low    INTEGER,
    peer_user_high   INTEGER,
    group_uuid       UUID,
    channel_uuid     UUID,
    pinned_by        INTEGER         NOT NULL,
    created_at       TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ(6),

    CONSTRAINT fk_message_pins_message
        FOREIGN KEY (message_id) REFERENCES messages (id),
    CONSTRAINT fk_message_pins_user
        FOREIGN KEY (pinned_by) REFERENCES user_relations (user_id)
);

CREATE INDEX IF NOT EXISTS idx_message_pins_message_id ON message_pins (message_id);
CREATE INDEX IF NOT EXISTS idx_message_pins_peer ON message_pins (peer_user_low, peer_user_high) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_message_pins_group ON message_pins (group_uuid) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_message_pins_channel ON message_pins (channel_uuid) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_message_pins_deleted_at ON message_pins (deleted_at);

-- Один пин одного сообщения в одном scope
CREATE UNIQUE INDEX IF NOT EXISTS idx_message_pins_unique_personal
    ON message_pins (message_id, peer_user_low, peer_user_high)
    WHERE deleted_at IS NULL AND peer_user_low IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_message_pins_unique_group
    ON message_pins (message_id, group_uuid)
    WHERE deleted_at IS NULL AND group_uuid IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_message_pins_unique_channel
    ON message_pins (message_id, channel_uuid)
    WHERE deleted_at IS NULL AND channel_uuid IS NOT NULL;

COMMENT ON TABLE message_pins IS 'Закреплённые сообщения в чатах';
