-- 数据库迁移任务与执行记录表

CREATE TABLE migration_tasks (
    id              VARCHAR(36)  PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    db_type         VARCHAR(20)  NOT NULL,          -- postgres | mysql
    source_host     VARCHAR(200) NOT NULL,
    source_port     INTEGER      NOT NULL,
    source_user     VARCHAR(100) NOT NULL,
    source_password TEXT         NOT NULL,          -- AES-256-GCM 加密
    db_names        TEXT         NOT NULL,
    target_host     VARCHAR(200) NOT NULL,
    target_port     INTEGER      NOT NULL,
    target_user     VARCHAR(100) NOT NULL,
    target_password TEXT         NOT NULL,          -- AES-256-GCM 加密
    mode            VARCHAR(30)  NOT NULL,          -- create_if_missing | overwrite
    description     VARCHAR(500) DEFAULT '',
    last_run_at     TIMESTAMP,
    last_run_status VARCHAR(30),
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP
);

CREATE INDEX idx_migration_tasks_deleted_at ON migration_tasks (deleted_at);
CREATE INDEX idx_migration_tasks_db_type    ON migration_tasks (db_type) WHERE deleted_at IS NULL;

CREATE TABLE migration_records (
    id          VARCHAR(36)  PRIMARY KEY,
    task_id     VARCHAR(36)  NOT NULL,
    task_name   VARCHAR(100) DEFAULT '',
    db_type     VARCHAR(20)  NOT NULL,
    mode        VARCHAR(30)  NOT NULL,
    status      VARCHAR(30)  NOT NULL,              -- running | success | partial_success | failed
    source_host VARCHAR(200) DEFAULT '',
    target_host VARCHAR(200) DEFAULT '',
    db_names    TEXT         DEFAULT '',
    summary     VARCHAR(500) DEFAULT '',
    error       TEXT         DEFAULT '',
    started_at  TIMESTAMP    NOT NULL,
    finished_at TIMESTAMP,
    duration    BIGINT       NOT NULL DEFAULT 0,
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_migration_records_task_id ON migration_records (task_id);
CREATE INDEX idx_migration_records_status  ON migration_records (status);

CREATE TABLE migration_record_items (
    id          VARCHAR(36)  PRIMARY KEY,
    record_id   VARCHAR(36)  NOT NULL,
    db_name     VARCHAR(100) NOT NULL,
    action      VARCHAR(30)  DEFAULT '',            -- created | overwritten | skipped
    status      VARCHAR(30)  NOT NULL,              -- success | failed | skipped
    message     TEXT         DEFAULT '',
    started_at  TIMESTAMP    NOT NULL,
    finished_at TIMESTAMP,
    duration    BIGINT       NOT NULL DEFAULT 0,
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_migration_record_items_record_id ON migration_record_items (record_id);
CREATE INDEX idx_migration_record_items_status    ON migration_record_items (status);
