DROP TRIGGER IF EXISTS trigger_populate_binary_processing_queue ON binaries;

DROP FUNCTION IF EXISTS populate_binary_processing_queue;

DROP TABLE IF EXISTS binary_processing_queue;
