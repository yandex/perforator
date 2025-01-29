CREATE TABLE IF NOT EXISTS binary_processing_queue(
	build_id TEXT NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
	status TEXT NOT NULL DEFAULT 'ready',
	processing_attempts INT NOT NULL DEFAULT 0,
	last_error TEXT,
	PRIMARY KEY(build_id)
);
CREATE INDEX IF NOT EXISTS queue_select_index ON binary_processing_queue(status, created_at);

CREATE OR REPLACE FUNCTION populate_binary_processing_queue() RETURNS TRIGGER AS
$populate_binary_processing_queue$
BEGIN
	IF (NEW.upload_status = 'uploaded') THEN
    	INSERT INTO binary_processing_queue(build_id)
		VALUES (NEW.build_id)
		ON CONFLICT DO NOTHING;
	END IF;

	RETURN NEW;
END;
$populate_binary_processing_queue$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trigger_populate_binary_processing_queue
AFTER UPDATE OF upload_status ON binaries
FOR EACH ROW
EXECUTE FUNCTION populate_binary_processing_queue();
