mise exec -- go run ./packages/engine/cmd/migrate --dry-run 
pending statements, not applied:

PRAGMA foreign_keys = off;
CREATE TABLE `blobs` (`id` text NOT NULL, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `content_hash` text NOT NULL, `size_bytes` integer NOT NULL, PRIMARY KEY (`id`));
CREATE UNIQUE INDEX `blob_content_hash` ON `blobs` (`content_hash`);
CREATE TABLE `objects` (`id` text NOT NULL, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `kind` text NOT NULL, `external_id` text NOT NULL, `first_seen_at` datetime NOT NULL, `last_seen_at` datetime NOT NULL, `deleted_at_source` datetime NULL, `source_id` text NOT NULL, PRIMARY KEY (`id`), CONSTRAINT `objects_sources_objects` FOREIGN KEY (`source_id`) REFERENCES `sources` (`id`) ON DELETE NO ACTION);
CREATE UNIQUE INDEX `object_source_id_external_id` ON `objects` (`source_id`, `external_id`);
CREATE TABLE `object_alias` (`id` text NOT NULL, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `source_id` text NOT NULL, `external_id` text NOT NULL, `object_id` text NOT NULL, PRIMARY KEY (`id`), CONSTRAINT `object_alias_objects_aliases` FOREIGN KEY (`object_id`) REFERENCES `objects` (`id`) ON DELETE NO ACTION);
CREATE UNIQUE INDEX `objectalias_source_id_external_id` ON `object_alias` (`source_id`, `external_id`);
CREATE TABLE `object_versions` (`id` text NOT NULL, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `restored_from_version_id` text NULL, `captured_at` datetime NOT NULL, `blob_id` text NOT NULL, `object_id` text NOT NULL, PRIMARY KEY (`id`), CONSTRAINT `object_versions_blobs_versions` FOREIGN KEY (`blob_id`) REFERENCES `blobs` (`id`) ON DELETE NO ACTION, CONSTRAINT `object_versions_objects_versions` FOREIGN KEY (`object_id`) REFERENCES `objects` (`id`) ON DELETE NO ACTION);
CREATE INDEX `objectversion_object_id_captured_at` ON `object_versions` (`object_id`, `captured_at`);
CREATE INDEX `objectversion_blob_id` ON `object_versions` (`blob_id`);
CREATE TABLE `sources` (`id` text NOT NULL, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `kind` text NOT NULL, `account_email` text NOT NULL, `status` text NOT NULL, PRIMARY KEY (`id`));
CREATE UNIQUE INDEX `source_kind_account_email` ON `sources` (`kind`, `account_email`);
PRAGMA foreign_keys = on;

