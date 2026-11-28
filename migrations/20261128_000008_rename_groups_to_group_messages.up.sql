-- groups was a message-to-group link table, not a group metadata table.
-- Rename it to make its purpose explicit while preserving all existing rows.
DO $$
BEGIN
    IF to_regclass('public.groups') IS NOT NULL
       AND to_regclass('public.group_messages') IS NULL THEN
        ALTER TABLE groups RENAME TO group_messages;
    END IF;
END $$;

-- Rename legacy indexes where present; PostgreSQL keeps indexes attached on table rename.
ALTER INDEX IF EXISTS idx_groups_message_id RENAME TO idx_group_messages_message_id;
ALTER INDEX IF EXISTS idx_groups_group_id RENAME TO idx_group_messages_group_id;
ALTER INDEX IF EXISTS idx_groups_deleted_at RENAME TO idx_group_messages_deleted_at;
ALTER INDEX IF EXISTS idx_groups_sender_id RENAME TO idx_group_messages_sender_id;
