CREATE TABLE IF NOT EXISTS message_reactions (
    id          SERIAL          PRIMARY KEY,
    message_id  INTEGER         NOT NULL,
    user_id     INTEGER         NOT NULL,
    emoji       VARCHAR(32)     NOT NULL,
    created_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ(6)  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ(6),

    CONSTRAINT fk_message_reactions_message
        FOREIGN KEY (message_id) REFERENCES messages (id),
    CONSTRAINT fk_message_reactions_user
        FOREIGN KEY (user_id) REFERENCES user_relations (user_id)
);

--Один и тот же emoji от одного пользователя на сообщение — один раз
CREATE UNIQUE INDEX IF NOT EXISTS idx_message_reactions_msg_user_emoji
    ON message_reactions (message_id, user_id, emoji)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_message_reactions_message_id
    ON message_reactions (message_id);

CREATE INDEX IF NOT EXISTS idx_message_reactions_user_id
    ON message_reactions (user_id);

CREATE INDEX IF NOT EXISTS idx_message_reactions_deleted_at
    ON message_reactions (deleted_at);

COMMENT ON TABLE message_reactions IS 'Реакции пользователей на сообщения (emoji)';
COMMENT ON COLUMN message_reactions.emoji IS 'Unicode emoji (до 8 code points / 32 байт)';
