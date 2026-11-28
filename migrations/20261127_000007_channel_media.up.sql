CREATE TABLE IF NOT EXISTS channel_media (
    id          SERIAL         PRIMARY KEY,
    channel_id  INTEGER        NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    file_uuid   UUID           NOT NULL,
    file_name   VARCHAR(255)   NOT NULL DEFAULT '',
    kind        VARCHAR(20)    NOT NULL DEFAULT 'audio',
    created_at  TIMESTAMPTZ(6) NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ(6) NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ(6)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_channel_media_unique
    ON channel_media(channel_id, file_uuid) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_channel_media_channel_id
    ON channel_media(channel_id) WHERE deleted_at IS NULL;
