-- 002: 将 Jenkins 配置从服务级别迁移到环境级别
-- jenkins_jobs: JSON 数组，如 [{"name":"构建","job":"folder/job-name"}]

-- 1. 为 service_envs 添加 jenkins_jobs 字段
ALTER TABLE service_envs
    ADD COLUMN IF NOT EXISTS jenkins_jobs JSONB NOT NULL DEFAULT '[]';

-- 2. 从 services 表迁移数据到对应环境（如果有的话）
UPDATE service_envs se
SET jenkins_jobs = jsonb_build_array(jsonb_build_object('name', '构建', 'job', s.jenkins_job))
FROM services s
WHERE se.service_id = s.id
  AND s.jenkins_job IS NOT NULL
  AND s.jenkins_job != ''
  AND se.jenkins_jobs = '[]';

-- 3. 移除 services 表上的 jenkins 相关字段
ALTER TABLE services DROP COLUMN IF EXISTS deploy_method;
ALTER TABLE services DROP COLUMN IF EXISTS jenkins_job;
