-- OpsHub MVP 数据库初始化 (PostgreSQL)
-- 建表顺序考虑外键依赖

-- =========================================
-- 1. 服务台账
-- =========================================

CREATE TABLE services (
    id            VARCHAR(36)  PRIMARY KEY,
    service_key   VARCHAR(64)  NOT NULL,
    service_name  VARCHAR(128) NOT NULL,
    owner_user_id VARCHAR(36)  NOT NULL DEFAULT '',
    repo_url      VARCHAR(512) NOT NULL DEFAULT '',
    tech_stack    VARCHAR(16)  NOT NULL DEFAULT 'go',        -- go | python
    runtime_type  VARCHAR(16)  NOT NULL DEFAULT 'docker',    -- systemd | docker | k8s
    deploy_type   VARCHAR(16)  NOT NULL DEFAULT 'internal',  -- external | internal | private
    sla_level     VARCHAR(8)   NOT NULL DEFAULT 'L3',        -- L1 | L2 | L3
    status        VARCHAR(16)  NOT NULL DEFAULT 'active',    -- active | inactive
    created_at    TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMP,

    CONSTRAINT uk_services_service_key UNIQUE (service_key)
);

CREATE INDEX idx_services_status ON services (status);
CREATE INDEX idx_services_owner  ON services (owner_user_id);

-- =========================================
-- 2. 服务环境
-- =========================================

CREATE TABLE service_envs (
    id                 VARCHAR(36)  PRIMARY KEY,
    service_id         VARCHAR(36)  NOT NULL REFERENCES services(id),
    env_code           VARCHAR(16)  NOT NULL,  -- dev | test | staging | prod
    cluster_name       VARCHAR(128) NOT NULL DEFAULT '',
    namespace          VARCHAR(128) NOT NULL DEFAULT '',
    access_endpoint    VARCHAR(512) NOT NULL DEFAULT '',
    healthcheck_url    VARCHAR(512) NOT NULL DEFAULT '',
    log_source_type    VARCHAR(16)  NOT NULL DEFAULT 'file',  -- loki | file | elasticsearch
    log_source_config  JSONB        NOT NULL DEFAULT '{}',
    created_at         TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMP,

    CONSTRAINT uk_service_envs UNIQUE (service_id, env_code)
);

-- =========================================
-- 3. 服务版本
-- =========================================

CREATE TABLE service_versions (
    id                VARCHAR(36)  PRIMARY KEY,
    service_id        VARCHAR(36)  NOT NULL REFERENCES services(id),
    version_no        VARCHAR(64)  NOT NULL,
    git_commit        VARCHAR(64)  NOT NULL DEFAULT '',
    artifact_uri      VARCHAR(512) NOT NULL DEFAULT '',
    build_pipeline_id VARCHAR(128) NOT NULL DEFAULT '',
    changelog         TEXT         NOT NULL DEFAULT '',
    created_by        VARCHAR(36)  NOT NULL DEFAULT '',
    created_at        TIMESTAMP    NOT NULL DEFAULT NOW(),

    CONSTRAINT uk_service_versions UNIQUE (service_id, version_no)
);

CREATE INDEX idx_service_versions_service ON service_versions (service_id, created_at DESC);

-- =========================================
-- 4. 发布记录
-- =========================================

CREATE TABLE release_records (
    id                VARCHAR(36)  PRIMARY KEY,
    service_id        VARCHAR(36)  NOT NULL REFERENCES services(id),
    env_id            VARCHAR(36)  NOT NULL REFERENCES service_envs(id),
    tenant_id         VARCHAR(36),           -- 可为空, 非私有化发布时为 NULL
    target_version_id VARCHAR(36)  NOT NULL REFERENCES service_versions(id),
    prev_version_id   VARCHAR(36),           -- 回滚来源
    release_type      VARCHAR(16)  NOT NULL DEFAULT 'deploy',  -- deploy | rollback
    strategy          VARCHAR(32)  NOT NULL DEFAULT 'default',
    status            VARCHAR(16)  NOT NULL DEFAULT 'pending', -- pending | running | success | failed | cancelled
    error_message     TEXT         NOT NULL DEFAULT '',
    operator_id       VARCHAR(36)  NOT NULL DEFAULT '',
    idempotency_key   VARCHAR(128),
    started_at        TIMESTAMP,
    ended_at          TIMESTAMP,
    created_at        TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMP    NOT NULL DEFAULT NOW(),

    CONSTRAINT uk_release_idempotency UNIQUE (idempotency_key)
);

CREATE INDEX idx_release_records_service ON release_records (service_id, created_at DESC);
CREATE INDEX idx_release_records_status  ON release_records (status);

-- =========================================
-- 5. 租户 (私有化客户)
-- =========================================

CREATE TABLE tenants (
    id              VARCHAR(36)  PRIMARY KEY,
    tenant_code     VARCHAR(64)  NOT NULL,
    tenant_name     VARCHAR(128) NOT NULL,
    license_type    VARCHAR(32)  NOT NULL DEFAULT 'standard', -- trial | standard | enterprise
    contract_start  DATE,
    contract_end    DATE,
    support_level   VARCHAR(16)  NOT NULL DEFAULT 'standard', -- basic | standard | premium
    upgrade_window  VARCHAR(64)  NOT NULL DEFAULT '',         -- e.g. "周三 02:00-06:00"
    status          VARCHAR(16)  NOT NULL DEFAULT 'active',
    created_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP,

    CONSTRAINT uk_tenants_code UNIQUE (tenant_code)
);

-- =========================================
-- 6. 租户-服务绑定
-- =========================================

CREATE TABLE tenant_service_bindings (
    id                  VARCHAR(36)  PRIMARY KEY,
    tenant_id           VARCHAR(36)  NOT NULL REFERENCES tenants(id),
    service_id          VARCHAR(36)  NOT NULL REFERENCES services(id),
    current_version_id  VARCHAR(36)  REFERENCES service_versions(id),
    pinned_version_flag BOOLEAN      NOT NULL DEFAULT FALSE,
    last_upgrade_at     TIMESTAMP,
    compat_check_status VARCHAR(16)  NOT NULL DEFAULT 'unknown', -- unknown | pass | fail
    created_at          TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP    NOT NULL DEFAULT NOW(),

    CONSTRAINT uk_tenant_service UNIQUE (tenant_id, service_id)
);

-- =========================================
-- 7. 配置项
-- =========================================

CREATE TABLE config_items (
    id             VARCHAR(36)  PRIMARY KEY,
    service_id     VARCHAR(36)  NOT NULL REFERENCES services(id),
    env_id         VARCHAR(36)  REFERENCES service_envs(id), -- NULL 表示全局 base 配置
    config_key     VARCHAR(256) NOT NULL,
    config_scope   VARCHAR(16)  NOT NULL DEFAULT 'base',     -- base | env | customer
    value_type     VARCHAR(16)  NOT NULL DEFAULT 'string',   -- string | int | bool | json | secret_ref
    default_value  TEXT         NOT NULL DEFAULT '',
    encrypted_flag BOOLEAN      NOT NULL DEFAULT FALSE,
    version_no     INT          NOT NULL DEFAULT 1,          -- 乐观锁
    created_by     VARCHAR(36)  NOT NULL DEFAULT '',
    created_at     TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP    NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMP,

    CONSTRAINT uk_config_items UNIQUE (service_id, env_id, config_key)
);

CREATE INDEX idx_config_items_service ON config_items (service_id);

-- =========================================
-- 8. 配置覆盖 (租户级)
-- =========================================

CREATE TABLE config_overrides (
    id              VARCHAR(36) PRIMARY KEY,
    tenant_id       VARCHAR(36) NOT NULL REFERENCES tenants(id),
    service_id      VARCHAR(36) NOT NULL REFERENCES services(id),
    env_id          VARCHAR(36) REFERENCES service_envs(id),
    config_item_id  VARCHAR(36) NOT NULL REFERENCES config_items(id),
    override_value  TEXT        NOT NULL DEFAULT '',
    version_no      INT         NOT NULL DEFAULT 1,
    effective_from  TIMESTAMP,
    effective_to    TIMESTAMP,
    updated_by      VARCHAR(36) NOT NULL DEFAULT '',
    created_at      TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP   NOT NULL DEFAULT NOW(),

    CONSTRAINT uk_config_overrides UNIQUE (tenant_id, config_item_id)
);

-- =========================================
-- 9. 告警事件
-- =========================================

CREATE TABLE alert_events (
    id                VARCHAR(36)  PRIMARY KEY,
    service_id        VARCHAR(36)  NOT NULL REFERENCES services(id),
    env_id            VARCHAR(36)  NOT NULL REFERENCES service_envs(id),
    tenant_id         VARCHAR(36),
    alert_source      VARCHAR(32)  NOT NULL DEFAULT 'custom',    -- prometheus | loki | custom
    alert_fingerprint VARCHAR(128) NOT NULL DEFAULT '',
    severity          VARCHAR(4)   NOT NULL DEFAULT 'P3',        -- P1 | P2 | P3 | P4
    title             VARCHAR(512) NOT NULL,
    content           TEXT         NOT NULL DEFAULT '',
    status            VARCHAR(16)  NOT NULL DEFAULT 'open',      -- open | acked | closed | suppressed
    first_seen_at     TIMESTAMP    NOT NULL DEFAULT NOW(),
    last_seen_at      TIMESTAMP    NOT NULL DEFAULT NOW(),
    assignee_user_id  VARCHAR(36)  NOT NULL DEFAULT '',
    created_at        TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMP    NOT NULL DEFAULT NOW(),

    CONSTRAINT uk_alert_fingerprint UNIQUE (alert_fingerprint)
);

CREATE INDEX idx_alert_events_service ON alert_events (service_id, status);
CREATE INDEX idx_alert_events_severity ON alert_events (severity, status);

-- =========================================
-- 10. 操作审计日志
-- =========================================

CREATE TABLE op_audit_logs (
    id               VARCHAR(36)  PRIMARY KEY,
    operator_id      VARCHAR(36)  NOT NULL DEFAULT '',
    module           VARCHAR(32)  NOT NULL,  -- service | release | config | alert | tenant
    action           VARCHAR(32)  NOT NULL,  -- create | update | delete | deploy | rollback | ack
    target_type      VARCHAR(64)  NOT NULL DEFAULT '',
    target_id        VARCHAR(36)  NOT NULL DEFAULT '',
    request_snapshot JSONB        NOT NULL DEFAULT '{}',
    result_code      INT          NOT NULL DEFAULT 0,
    ip               VARCHAR(45)  NOT NULL DEFAULT '',
    created_at       TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_op_audit_logs_module ON op_audit_logs (module, created_at DESC);
CREATE INDEX idx_op_audit_logs_target ON op_audit_logs (target_type, target_id);
