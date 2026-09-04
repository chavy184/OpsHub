CREATE TABLE IF NOT EXISTS image_sync_records (
    id VARCHAR(36) PRIMARY KEY,
    source_host_id VARCHAR(36) NOT NULL,
    source_host_name VARCHAR(100),
    target_host_id VARCHAR(36) NOT NULL,
    target_host_name VARCHAR(100),
    image VARCHAR(500) NOT NULL,
    mode VARCHAR(30) NOT NULL,
    status VARCHAR(30) NOT NULL,
    source_image_id VARCHAR(200),
    target_image_id VARCHAR(200),
    image_size BIGINT DEFAULT 0,
    summary VARCHAR(500),
    error TEXT,
    started_at TIMESTAMP NOT NULL,
    finished_at TIMESTAMP NULL,
    duration BIGINT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_image_sync_records_source_host_id ON image_sync_records (source_host_id);
CREATE INDEX IF NOT EXISTS idx_image_sync_records_target_host_id ON image_sync_records (target_host_id);
CREATE INDEX IF NOT EXISTS idx_image_sync_records_status ON image_sync_records (status);
CREATE INDEX IF NOT EXISTS idx_image_sync_records_image ON image_sync_records (image);
