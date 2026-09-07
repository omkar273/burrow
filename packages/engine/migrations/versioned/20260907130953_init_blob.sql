-- Create "blobs" table
CREATE TABLE `blobs` (
  `id` text NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `content_hash` text NOT NULL,
  `size_bytes` integer NOT NULL,
  PRIMARY KEY (`id`)
);
-- Create index "blob_content_hash" to table: "blobs"
CREATE UNIQUE INDEX `blob_content_hash` ON `blobs` (`content_hash`);
