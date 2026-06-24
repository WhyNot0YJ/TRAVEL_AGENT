ALTER TABLE travel_tasks
  ADD COLUMN request_id VARCHAR(128) NULL AFTER id,
  ADD KEY idx_travel_tasks_request_id (request_id);
