-- 备份任务与备份记录表

CREATE TABLE backup_tasks (
    id              VARCHAR(36)  PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    backup_type     VARCHAR(20)  NOT NULL,          -- postgres | mysql
    cron_expr       VARCHAR(50)  NOT NULL,
    enabled         BOOLEAN      NOT NULL DEFAULT TRUE,
    db_host         VARCHAR(200) NOT NULL,
    db_port         INTEGER      NOT NULL,
    db_user         VARCHAR(100) NOT NULL,
    db_password     TEXT         NOT NULL,          -- AES-256-GCM 加密
    db_name         VARCHAR(100) DEFAULT '',
    target_host_id  VARCHAR(36)  NOT NULL,
    target_path     VARCHAR(500) NOT NULL,
    retention_days  INTEGER      NOT NULL DEFAULT 10,
    description     VARCHAR(500) DEFAULT '',
    last_run_at     TIMESTAMP,
    last_run_status VARCHAR(20),
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP
);

CREATE INDEX idx_backup_tasks_enabled    ON backup_tasks (enabled) WHERE deleted_at IS NULL;
CREATE INDEX idx_backup_tasks_deleted_at ON backup_tasks (deleted_at);

CREATE TABLE backup_records (
    id          VARCHAR(36)  PRIMARY KEY,
    task_id     VARCHAR(36)  NOT NULL,
    task_name   VARCHAR(100) DEFAULT '',
    status      VARCHAR(20)  NOT NULL,              -- pending | running | success | failed
    file_name   VARCHAR(500) DEFAULT '',
    file_size   BIGINT       NOT NULL DEFAULT 0,
    duration    BIGINT       NOT NULL DEFAULT 0,    -- 秒
    error       TEXT         DEFAULT '',
    started_at  TIMESTAMP    NOT NULL,
    finished_at TIMESTAMP,
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_backup_records_task_id ON backup_records (task_id);
CREATE INDEX idx_backup_records_status  ON backup_records (status);
