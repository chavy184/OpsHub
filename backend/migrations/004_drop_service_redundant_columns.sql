-- 移除 services 表中已下线的字段。该表仅保留身份/名称/仓库/负责人。
ALTER TABLE services DROP COLUMN IF EXISTS tech_stack;
ALTER TABLE services DROP COLUMN IF EXISTS runtime_type;
ALTER TABLE services DROP COLUMN IF EXISTS deploy_type;
ALTER TABLE services DROP COLUMN IF EXISTS sla_level;
ALTER TABLE services DROP COLUMN IF EXISTS status;
ALTER TABLE services DROP COLUMN IF EXISTS default_branch;
ALTER TABLE services DROP COLUMN IF EXISTS build_command;
ALTER TABLE services DROP COLUMN IF EXISTS start_command;
ALTER TABLE services DROP COLUMN IF EXISTS stop_command;
ALTER TABLE services DROP COLUMN IF EXISTS docker_image;
ALTER TABLE services DROP COLUMN IF EXISTS dockerfile_path;
ALTER TABLE services DROP COLUMN IF EXISTS work_dir;
ALTER TABLE services DROP COLUMN IF EXISTS jenkins_job_url;
