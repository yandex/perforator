ALTER TABLE tasks
ADD CONSTRAINT unique_idempotency_key UNIQUE (idempotency_key);