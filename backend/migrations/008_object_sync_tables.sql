CREATE TABLE IF NOT EXISTS object_sync_tasks (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    source_endpoint VARCHAR(300) NOT NULL,
    source_region VARCHAR(100),
    source_bucket VARCHAR(200) NOT NULL,
    source_path VARCHAR(1000),
    source_access_key TEXT NOT NULL,
    source_secret_key TEXT NOT NULL,
    source_use_ssl BOOLEAN DEFAULT TRUE,
    target_endpoint VARCHAR(300) NOT NULL,
    target_region VARCHAR(100),
    target_bucket VARCHAR(200) NOT NULL,
    target_path VARCHAR(1000),
    target_access_key TEXT NOT NULL,
    target_secret_key TEXT NOT NULL,
    target_use_ssl BOOLEAN DEFAULT TRUE,
    mode VARCHAR(30) NOT NULL,
    description VARCHAR(500),
    last_run_at TIMESTAMP NULL,
    last_run_status VARCHAR(30),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX IF NOT EXISTS idx_object_sync_tasks_deleted_at ON object_sync_tasks (deleted_at);

CREATE TABLE IF NOT EXISTS object_sync_records (
    id VARCHAR(36) PRIMARY KEY,
    task_id VARCHAR(36) NOT NULL,
    task_name VARCHAR(100),
    mode VARCHAR(30) NOT NULL,
    status VARCHAR(30) NOT NULL,
    source_bucket VARCHAR(200),
    source_path VARCHAR(1000),
    target_bucket VARCHAR(200),
    target_path VARCHAR(1000),
    object_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    skipped_count INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0,
    bytes_total BIGINT DEFAULT 0,
    summary VARCHAR(500),
    error TEXT,
    started_at TIMESTAMP NOT NULL,
    finished_at TIMESTAMP NULL,
    duration BIGINT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_object_sync_records_task_id ON object_sync_records (task_id);
CREATE INDEX IF NOT EXISTS idx_object_sync_records_status ON object_sync_records (status);

CREATE TABLE IF NOT EXISTS object_sync_record_items (
    id VARCHAR(36) PRIMARY KEY,
    record_id VARCHAR(36) NOT NULL,
    source_key VARCHAR(1000) NOT NULL,
    target_key VARCHAR(1000) NOT NULL,
    size BIGINT DEFAULT 0,
    etag VARCHAR(200),
    action VARCHAR(30),
    status VARCHAR(30) NOT NULL,
    message TEXT,
    started_at TIMESTAMP NOT NULL,
    finished_at TIMESTAMP NULL,
    duration BIGINT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_object_sync_record_items_record_id ON object_sync_record_items (record_id);
CREATE INDEX IF NOT EXISTS idx_object_sync_record_items_status ON object_sync_record_items (status);
