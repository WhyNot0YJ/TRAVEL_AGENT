-- Backend performance persistence upgrade.
-- Run after 001_travel_persistence.sql, 002_observability_request_id.sql and
-- 003_users_and_plan_library.sql.

ALTER TABLE travel_tasks
  ADD COLUMN IF NOT EXISTS planner_type VARCHAR(64) NOT NULL DEFAULT 'unknown' AFTER status,
  ADD COLUMN IF NOT EXISTS agent_mode VARCHAR(32) NOT NULL DEFAULT 'quick' AFTER planner_type,
  ADD COLUMN IF NOT EXISTS test_mode BOOLEAN NOT NULL DEFAULT FALSE AFTER agent_mode,
  ADD COLUMN IF NOT EXISTS attempt INT NOT NULL DEFAULT 1 AFTER test_mode;

CREATE TABLE IF NOT EXISTS travel_task_requests (
  task_id VARCHAR(64) NOT NULL,
  request_hash VARCHAR(128) NOT NULL,
  request_json JSON NOT NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  PRIMARY KEY (task_id),
  KEY idx_travel_task_requests_hash (request_hash),
  CONSTRAINT fk_travel_task_requests_task
    FOREIGN KEY (task_id) REFERENCES travel_tasks(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS travel_plan_results (
  task_id VARCHAR(64) NOT NULL,
  result_version INT NOT NULL DEFAULT 1,
  plan_json JSON NOT NULL,
  budget_total DECIMAL(12,2) NOT NULL DEFAULT 0,
  day_count INT NOT NULL DEFAULT 0,
  warning_count INT NOT NULL DEFAULT 0,
  generated_duration_ms BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  PRIMARY KEY (task_id),
  CONSTRAINT fk_travel_plan_results_task
    FOREIGN KEY (task_id) REFERENCES travel_tasks(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS travel_planner_runs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  task_id VARCHAR(64) NOT NULL,
  planner_type VARCHAR(64) NOT NULL,
  worker_id VARCHAR(128) NULL,
  attempt INT NOT NULL DEFAULT 1,
  started_at DATETIME(6) NOT NULL,
  finished_at DATETIME(6) NULL,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL,
  fallback_used BOOLEAN NOT NULL DEFAULT FALSE,
  fallback_reason VARCHAR(512) NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY ux_travel_planner_runs_task_attempt (task_id, attempt),
  KEY idx_travel_planner_runs_task_created (task_id, created_at),
  KEY idx_travel_planner_runs_status_finished (status, finished_at),
  CONSTRAINT fk_travel_planner_runs_task
    FOREIGN KEY (task_id) REFERENCES travel_tasks(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS travel_node_traces (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  task_id VARCHAR(64) NOT NULL,
  run_id BIGINT UNSIGNED NULL,
  node_name VARCHAR(128) NOT NULL,
  node_status VARCHAR(32) NOT NULL,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  warning TEXT NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_travel_node_traces_task_created (task_id, created_at),
  KEY idx_travel_node_traces_run_created (run_id, created_at),
  CONSTRAINT fk_travel_node_traces_task
    FOREIGN KEY (task_id) REFERENCES travel_tasks(id)
    ON DELETE CASCADE,
  CONSTRAINT fk_travel_node_traces_run
    FOREIGN KEY (run_id) REFERENCES travel_planner_runs(id)
    ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS travel_error_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  request_id VARCHAR(128) NULL,
  trace_id VARCHAR(128) NULL,
  task_id VARCHAR(64) NULL,
  run_id BIGINT UNSIGNED NULL,
  component VARCHAR(64) NOT NULL,
  operation VARCHAR(64) NOT NULL,
  error_category VARCHAR(64) NOT NULL,
  error_code VARCHAR(64) NOT NULL,
  retryable BOOLEAN NOT NULL DEFAULT FALSE,
  attempt INT NOT NULL DEFAULT 1,
  message TEXT NOT NULL,
  stack_hash VARCHAR(128) NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_travel_error_logs_trace_created (trace_id, created_at),
  KEY idx_travel_error_logs_task_created (task_id, created_at),
  KEY idx_travel_error_logs_category_created (error_category, created_at),
  KEY idx_travel_error_logs_request_created (request_id, created_at),
  CONSTRAINT fk_travel_error_logs_task
    FOREIGN KEY (task_id) REFERENCES travel_tasks(id)
    ON DELETE SET NULL,
  CONSTRAINT fk_travel_error_logs_run
    FOREIGN KEY (run_id) REFERENCES travel_planner_runs(id)
    ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
