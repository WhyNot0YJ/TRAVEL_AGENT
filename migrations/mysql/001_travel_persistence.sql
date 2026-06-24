CREATE TABLE IF NOT EXISTS travel_tasks (
  id VARCHAR(64) NOT NULL,
  request_hash VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL,
  request_json JSON NOT NULL,
  error_text TEXT NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY ux_travel_tasks_request_hash (request_hash),
  KEY idx_travel_tasks_status_updated_at (status, updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS travel_plans (
  task_id VARCHAR(64) NOT NULL,
  plan_json JSON NOT NULL,
  budget_total DECIMAL(12,2) NOT NULL DEFAULT 0,
  day_count INT NOT NULL DEFAULT 0,
  warning_count INT NOT NULL DEFAULT 0,
  updated_at DATETIME(6) NOT NULL,
  PRIMARY KEY (task_id),
  CONSTRAINT fk_travel_plans_task
    FOREIGN KEY (task_id) REFERENCES travel_tasks(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS planner_runs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  task_id VARCHAR(64) NOT NULL,
  planner_type VARCHAR(64) NOT NULL,
  prompt_version VARCHAR(64) NULL,
  tool_mode VARCHAR(32) NULL,
  started_at DATETIME(6) NOT NULL,
  finished_at DATETIME(6) NULL,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL,
  fallback_used BOOLEAN NOT NULL DEFAULT FALSE,
  fallback_reason VARCHAR(512) NULL,
  prompt_tokens INT NULL,
  completion_tokens INT NULL,
  total_tokens INT NULL,
  PRIMARY KEY (id),
  KEY idx_planner_runs_task_id (task_id),
  CONSTRAINT fk_planner_runs_task
    FOREIGN KEY (task_id) REFERENCES travel_tasks(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS planner_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  task_id VARCHAR(64) NOT NULL,
  run_id BIGINT UNSIGNED NULL,
  node_name VARCHAR(128) NULL,
  tool_name VARCHAR(64) NULL,
  provider VARCHAR(64) NULL,
  status VARCHAR(32) NOT NULL,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  fallback_reason VARCHAR(512) NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_planner_events_task_id (task_id),
  KEY idx_planner_events_run_id (run_id),
  CONSTRAINT fk_planner_events_task
    FOREIGN KEY (task_id) REFERENCES travel_tasks(id)
    ON DELETE CASCADE,
  CONSTRAINT fk_planner_events_run
    FOREIGN KEY (run_id) REFERENCES planner_runs(id)
    ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
