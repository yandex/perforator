ALTER TABLE binaries
    DROP COLUMN IF EXISTS compression,
    DROP COLUMN IF EXISTS uncompressed_size;
