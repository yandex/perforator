-- Stop new lease consumers before rollback. Free named rows can be recreated
-- by the old implementation; held rows must survive rollback.
DELETE FROM leases WHERE holder IS NULL OR expires_at IS NULL;
ALTER TABLE leases
    ALTER COLUMN holder SET NOT NULL,
    ALTER COLUMN expires_at SET NOT NULL;
