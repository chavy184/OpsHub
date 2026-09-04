-- 003_multi_gpu.sql
-- 主机指标快照新增多 GPU JSON 字段（保留单 GPU 字段向后兼容）

ALTER TABLE host_metric_snapshots
  ADD COLUMN IF NOT EXISTS gpus_json JSONB NOT NULL DEFAULT '[]';

COMMENT ON COLUMN host_metric_snapshots.gpus_json IS '多 GPU 信息数组：[{index,name,usage,mem_usage,temp}]';
