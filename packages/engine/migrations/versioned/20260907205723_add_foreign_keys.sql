-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_objects" table
CREATE TABLE `new_objects` (
  `id` text NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `kind` text NOT NULL,
  `external_id` text NOT NULL,
  `first_seen_at` datetime NOT NULL,
  `last_seen_at` datetime NOT NULL,
  `deleted_at_source` datetime NULL,
  `source_id` text NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `objects_sources_objects` FOREIGN KEY (`source_id`) REFERENCES `sources` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Copy rows from old table "objects" to new temporary table "new_objects"
INSERT INTO `new_objects` (`id`, `created_at`, `updated_at`, `kind`, `external_id`, `first_seen_at`, `last_seen_at`, `deleted_at_source`, `source_id`) SELECT `id`, `created_at`, `updated_at`, `kind`, `external_id`, `first_seen_at`, `last_seen_at`, `deleted_at_source`, `source_id` FROM `objects`;
-- Drop "objects" table after copying rows
DROP TABLE `objects`;
-- Rename temporary table "new_objects" to "objects"
ALTER TABLE `new_objects` RENAME TO `objects`;
-- Create index "object_source_id_external_id" to table: "objects"
CREATE UNIQUE INDEX `object_source_id_external_id` ON `objects` (`source_id`, `external_id`);
-- Create "new_object_alias" table
CREATE TABLE `new_object_alias` (
  `id` text NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `source_id` text NOT NULL,
  `external_id` text NOT NULL,
  `object_id` text NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `object_alias_objects_aliases` FOREIGN KEY (`object_id`) REFERENCES `objects` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Copy rows from old table "object_alias" to new temporary table "new_object_alias"
INSERT INTO `new_object_alias` (`id`, `created_at`, `updated_at`, `source_id`, `external_id`, `object_id`) SELECT `id`, `created_at`, `updated_at`, `source_id`, `external_id`, `object_id` FROM `object_alias`;
-- Drop "object_alias" table after copying rows
DROP TABLE `object_alias`;
-- Rename temporary table "new_object_alias" to "object_alias"
ALTER TABLE `new_object_alias` RENAME TO `object_alias`;
-- Create index "objectalias_source_id_external_id" to table: "object_alias"
CREATE UNIQUE INDEX `objectalias_source_id_external_id` ON `object_alias` (`source_id`, `external_id`);
-- Create "new_object_versions" table
CREATE TABLE `new_object_versions` (
  `id` text NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `restored_from_version_id` text NULL,
  `captured_at` datetime NOT NULL,
  `blob_id` text NOT NULL,
  `object_id` text NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `object_versions_objects_versions` FOREIGN KEY (`object_id`) REFERENCES `objects` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `object_versions_blobs_versions` FOREIGN KEY (`blob_id`) REFERENCES `blobs` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Copy rows from old table "object_versions" to new temporary table "new_object_versions"
INSERT INTO `new_object_versions` (`id`, `created_at`, `updated_at`, `restored_from_version_id`, `captured_at`, `blob_id`, `object_id`) SELECT `id`, `created_at`, `updated_at`, `restored_from_version_id`, `captured_at`, `blob_id`, `object_id` FROM `object_versions`;
-- Drop "object_versions" table after copying rows
DROP TABLE `object_versions`;
-- Rename temporary table "new_object_versions" to "object_versions"
ALTER TABLE `new_object_versions` RENAME TO `object_versions`;
-- Create index "objectversion_object_id_captured_at" to table: "object_versions"
CREATE INDEX `objectversion_object_id_captured_at` ON `object_versions` (`object_id`, `captured_at`);
-- Create index "objectversion_blob_id" to table: "object_versions"
CREATE INDEX `objectversion_blob_id` ON `object_versions` (`blob_id`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
