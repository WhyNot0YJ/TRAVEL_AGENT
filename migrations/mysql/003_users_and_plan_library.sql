-- Stage 21: users, sessions, user plan library, conversation archives, public plans.
-- Idempotent CREATE TABLE IF NOT EXISTS so it can be re-run safely.

CREATE TABLE IF NOT EXISTS users (
  id VARCHAR(64) NOT NULL,
  email VARCHAR(190) NOT NULL,
  display_name VARCHAR(80) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY ux_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS user_sessions (
  id VARCHAR(64) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  token_hash VARCHAR(128) NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  created_at DATETIME(6) NOT NULL,
  revoked_at DATETIME(6) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY ux_user_sessions_token_hash (token_hash),
  KEY idx_user_sessions_user_expires (user_id, expires_at),
  CONSTRAINT fk_user_sessions_user
    FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS user_plans (
  id VARCHAR(64) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NULL,
  source_public_plan_id VARCHAR(64) NULL,
  title VARCHAR(160) NOT NULL,
  note TEXT NULL,
  summary TEXT NULL,
  tags_json JSON NULL,
  plan_json JSON NOT NULL,
  destination_city VARCHAR(120) NOT NULL DEFAULT '',
  days INT NOT NULL DEFAULT 0,
  visibility VARCHAR(16) NOT NULL DEFAULT 'private',
  publish_status VARCHAR(16) NOT NULL DEFAULT 'draft',
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  deleted_at DATETIME(6) NULL,
  PRIMARY KEY (id),
  KEY idx_user_plans_user_updated (user_id, updated_at),
  KEY idx_user_plans_user_deleted (user_id, deleted_at),
  KEY idx_user_plans_destination (destination_city),
  KEY idx_user_plans_visibility_publish_updated (visibility, publish_status, updated_at),
  KEY idx_user_plans_task (task_id),
  CONSTRAINT fk_user_plans_user
    FOREIGN KEY (user_id) REFERENCES users(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS plan_conversation_archives (
  id VARCHAR(64) NOT NULL,
  plan_id VARCHAR(64) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NULL,
  brief_json JSON NULL,
  messages_json JSON NULL,
  events_json JSON NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_plan_archives_plan (plan_id),
  KEY idx_plan_archives_user (user_id),
  CONSTRAINT fk_plan_archives_plan
    FOREIGN KEY (plan_id) REFERENCES user_plans(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS public_plans (
  id VARCHAR(64) NOT NULL,
  plan_id VARCHAR(64) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  title VARCHAR(160) NOT NULL,
  summary TEXT NULL,
  tags_json JSON NULL,
  plan_json JSON NOT NULL,
  destination_city VARCHAR(120) NOT NULL DEFAULT '',
  days INT NOT NULL DEFAULT 0,
  status VARCHAR(16) NOT NULL DEFAULT 'published',
  view_count BIGINT NOT NULL DEFAULT 0,
  save_count BIGINT NOT NULL DEFAULT 0,
  copy_count BIGINT NOT NULL DEFAULT 0,
  hot_score BIGINT NOT NULL DEFAULT 0,
  published_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY ux_public_plans_plan (plan_id),
  KEY idx_public_plans_status_hot (status, hot_score, published_at),
  KEY idx_public_plans_status_published (status, published_at),
  KEY idx_public_plans_destination (destination_city, status),
  KEY idx_public_plans_user (user_id, published_at),
  CONSTRAINT fk_public_plans_plan
    FOREIGN KEY (plan_id) REFERENCES user_plans(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS public_plan_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  public_plan_id VARCHAR(64) NOT NULL,
  user_id VARCHAR(64) NULL,
  event_type VARCHAR(16) NOT NULL,
  client_hash VARCHAR(128) NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_public_plan_events_plan (public_plan_id),
  KEY idx_public_plan_events_type_created (event_type, created_at),
  CONSTRAINT fk_public_plan_events_plan
    FOREIGN KEY (public_plan_id) REFERENCES public_plans(id)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- travel_tasks gains optional user_id so saved plans can be attributed to the
-- creator. Older anonymous tasks remain NULL and stay readable; saving now
-- requires user_id IS NULL OR user_id = ?.
ALTER TABLE travel_tasks
  ADD COLUMN user_id VARCHAR(64) NULL AFTER request_id,
  ADD KEY idx_travel_tasks_user_id (user_id);

CREATE TABLE IF NOT EXISTS analytics_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  event_name VARCHAR(64) NOT NULL,
  request_id VARCHAR(128) NULL,
  user_id VARCHAR(64) NULL,
  plan_id VARCHAR(64) NULL,
  public_plan_id VARCHAR(64) NULL,
  task_id VARCHAR(64) NULL,
  destination_city VARCHAR(120) NULL,
  days INT NULL,
  source VARCHAR(64) NULL,
  payload_json JSON NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (id),
  KEY idx_analytics_events_name_created (event_name, created_at),
  KEY idx_analytics_events_user_created (user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
