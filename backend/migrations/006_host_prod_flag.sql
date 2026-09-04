-- 006: 主机增加 Prod 标记，用于保护数据库迁移、对象同步等高危目标。

ALTER TABLE hosts
  ADD COLUMN IF NOT EXISTS is_prod BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_hosts_is_prod ON hosts(is_prod);
