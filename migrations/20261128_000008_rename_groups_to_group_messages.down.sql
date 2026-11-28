-- Reverse the table rename without changing or deleting message links.
DO $$
BEGIN
    IF to_regclass('public.group_messages') IS NOT NULL
       AND to_regclass('public.groups') IS NULL THEN
        ALTER TABLE group_messages RENAME TO groups;
    END IF;
END $$;

ALTER INDEX IF EXISTS idx_group_messages_message_id RENAME TO idx_groups_message_id;
ALTER INDEX IF EXISTS idx_group_messages_group_id RENAME TO idx_groups_group_id;
ALTER INDEX IF EXISTS idx_group_messages_deleted_at RENAME TO idx_groups_deleted_at;
ALTER INDEX IF EXISTS idx_group_messages_sender_id RENAME TO idx_groups_sender_id;
