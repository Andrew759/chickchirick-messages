-- Откат полной схемы chickchirick-messages (обратный порядок зависимостей)

DROP TABLE IF EXISTS e2ee_one_time_prekeys;
DROP TABLE IF EXISTS e2ee_identities;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS message_meta;
DROP TABLE IF EXISTS statuses;
DROP TABLE IF EXISTS deleteds;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS personals;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS user_relations;
