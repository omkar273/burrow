-- Create "objects" table
CREATE TABLE `objects` (
  `id` text NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `source_id` text NOT NULL,
  `kind` text NOT NULL,
  `external_id` text NOT NULL,
  `first_seen_at` datetime NOT NULL,
  `last_seen_at` datetime NOT NULL,
  `deleted_at_source` datetime NULL,
  PRIMARY KEY (`id`)
);
-- Create index "object_source_id_external_id" to table: "objects"
CREATE UNIQUE INDEX `object_source_id_external_id` ON `objects` (`source_id`, `external_id`);
-- Create "object_alias" table
CREATE TABLE `object_alias` (
  `id` text NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `object_id` text NOT NULL,
  `source_id` text NOT NULL,
  `external_id` text NOT NULL,
  PRIMARY KEY (`id`)
);
-- Create index "objectalias_source_id_external_id" to table: "object_alias"
CREATE UNIQUE INDEX `objectalias_source_id_external_id` ON `object_alias` (`source_id`, `external_id`);
-- Create "object_versions" table
CREATE TABLE `object_versions` (
  `id` text NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `object_id` text NOT NULL,
  `blob_id` text NOT NULL,
  `restored_from_version_id` text NULL,
  `captured_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
);
-- Create index "objectversion_object_id_captured_at" to table: "object_versions"
CREATE INDEX `objectversion_object_id_captured_at` ON `object_versions` (`object_id`, `captured_at`);
-- Create index "objectversion_blob_id" to table: "object_versions"
CREATE INDEX `objectversion_blob_id` ON `object_versions` (`blob_id`);
-- Create "sources" table
CREATE TABLE `sources` (
  `id` text NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `kind` text NOT NULL,
  `account_email` text NOT NULL,
  `status` text NOT NULL,
  PRIMARY KEY (`id`)
);
-- Create index "source_kind_account_email" to table: "sources"
CREATE UNIQUE INDEX `source_kind_account_email` ON `sources` (`kind`, `account_email`);
