// OpsHub 前端类型定义 — 与后端 DTO 严格对齐

// ============================
// 通用
// ============================

export interface ApiResponse<T = unknown> {
  code: number
  msg: string
  data: T
}

export interface PageData<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface PageParams {
  page: number
  page_size: number
  keyword?: string
}

export interface LoginPayload {
  username: string
  password: string
}

export interface LoginResult {
  token: string
  username: string
  expires_at: string
}

// ============================
// 服务台账
// ============================

export type TechStack = 'go' | 'python'
export type RuntimeType = 'systemd' | 'docker' | 'file' | 'other'
export type DeployType = 'external' | 'internal' | 'private'
export type ServiceStatus = 'active' | 'inactive'

export interface Service {
  id: string
  service_key: string
  service_name: string
  owner_user_id: string
  repo_url: string
  created_at: string
  updated_at: string
}

export interface CreateServicePayload {
  service_key: string
  service_name: string
  owner_user_id?: string
  repo_url?: string
}

export interface UpdateServicePayload {
  service_name?: string
  owner_user_id?: string
  repo_url?: string
}

export interface ServiceQueryParams extends PageParams {
  // 预留仅关键字搜索
}

// ============================
// 服务环境
// ============================

export interface ServiceEnv {
  id: string
  service_id: string
  env_code: string
  cluster_name: string
  namespace: string
  access_endpoint: string
  healthcheck_url: string
  log_source_type: string
  log_source_config: string
  host_id: string
  deploy_path: string
  env_vars: string
  health_check_enabled: boolean
  health_check_interval: number
  health_check_timeout: number
  health_check_success_codes: string
  health_status: string
  health_last_checked_at: string | null
  health_last_message: string
  jenkins_jobs: string
}

export interface CreateEnvPayload {
  env_code: string
  cluster_name?: string
  namespace?: string
  access_endpoint?: string
  healthcheck_url?: string
  log_source_type?: string
  log_source_config?: string
  host_id?: string
  deploy_path?: string
  health_check_enabled?: boolean
  jenkins_jobs?: string
}

export interface UpdateEnvPayload {
  host_id?: string
  deploy_path?: string
  access_endpoint?: string
  healthcheck_url?: string
  log_source_type?: string
  log_source_config?: string
  health_check_interval?: number
  health_check_timeout?: number
  health_check_success_codes?: string
  health_check_enabled?: boolean
  jenkins_jobs?: string
}

// ============================
// 发布记录
// ============================

export type ReleaseType = 'deploy' | 'rollback'
export type ReleaseStatus = 'pending' | 'running' | 'success' | 'failed' | 'cancelled'

export interface ReleaseRecord {
  id: string
  service_id: string
  env_id: string
  tenant_id: string
  target_version_id: string
  prev_version_id: string
  release_type: ReleaseType
  strategy: string
  status: ReleaseStatus
  error_message: string
  operator_id: string
  jenkins_params: string
  jenkins_build_no: number
  started_at: string | null
  ended_at: string | null
  created_at: string
  env_code: string
}

export interface CreateReleasePayload {
  service_id: string
  env_id?: string
  tenant_id?: string
  target_version_id?: string
  strategy?: string
  jenkins_params?: Record<string, string>
  force_prod_target?: boolean
  admin_password?: string
  idempotency_key?: string
}

export interface ReleaseQueryParams extends PageParams {
  service_id?: string
  env_id?: string
  tenant_id?: string
  status?: ReleaseStatus
  release_type?: ReleaseType
}

// ============================
// 租户
// ============================

export type TenantStatus = 'active' | 'inactive'

export interface Tenant {
  id: string
  tenant_code: string
  tenant_name: string
  license_type: string
  contract_start: string | null
  contract_end: string | null
  support_level: string
  upgrade_window: string
  status: TenantStatus
  created_at: string
  updated_at: string
}

// ============================
// 告警
// ============================

export type Severity = 'P1' | 'P2' | 'P3' | 'P4'
export type AlertStatus = 'open' | 'acked' | 'closed' | 'suppressed'

export interface AlertEvent {
  id: string
  service_id: string
  env_id: string
  tenant_id: string
  alert_source: string
  alert_fingerprint: string
  severity: Severity
  title: string
  content: string
  status: AlertStatus
  first_seen_at: string
  last_seen_at: string
  assignee_user_id: string
  created_at: string
}

// ============================
// 日志检索
// ============================

export interface LogEntry {
  timestamp: string
  level: string
  message: string
  service: string
  env: string
  trace_id: string
  fields: Record<string, string>
}

export interface LogSearchParams {
  service_id: string
  env_id?: string
  keyword?: string
  level?: string
  trace_id?: string
  start_time?: string
  end_time?: string
  page?: number
  page_size?: number
}

// ============================
// 发布步骤日志
// ============================

export type StepStatus = 'pending' | 'running' | 'success' | 'failed' | 'skipped'

export interface ReleaseStepLog {
  id: string
  release_id: string
  step_order: number
  step_name: string
  step_status: StepStatus
  started_at: string | null
  ended_at: string | null
  duration_ms: number
  output: string
  error_output: string
  created_at: string
}

// ============================
// 凭证
// ============================

export type CredType = 'ssh_key' | 'ssh_password' | 'token'

export interface Credential {
  id: string
  name: string
  cred_type: CredType
  fingerprint: string
  description: string
  created_by: string
  created_at: string
  updated_at: string
}

export interface CreateCredentialPayload {
  name: string
  cred_type: CredType
  secret_data: string
  passphrase?: string
  description?: string
}

export interface UpdateCredentialPayload {
  name?: string
  secret_data?: string
  passphrase?: string
  description?: string
}

export interface CredentialQueryParams extends PageParams {}

// ============================
// 主机
// ============================

export type AgentStatus = 'online' | 'offline' | 'unknown'

export interface Host {
  id: string
  name: string
  host_address: string
  ssh_port: number
  username: string
  credential_id: string
  is_prod: boolean
  labels: Record<string, string>
  os_info: string
  agent_status: AgentStatus
  last_heartbeat: string | null
  description: string
  created_at: string
  updated_at: string
}

export interface CreateHostPayload {
  name: string
  host_address: string
  ssh_port?: number
  username?: string
  credential_id?: string
  is_prod?: boolean
  labels?: Record<string, string>
  description?: string
}

export interface UpdateHostPayload {
  name?: string
  host_address?: string
  ssh_port?: number
  username?: string
  credential_id?: string
  is_prod?: boolean
  labels?: Record<string, string>
  description?: string
}

export interface HostQueryParams extends PageParams {
  agent_status?: AgentStatus
}

export interface TestConnectionResult {
  success: boolean
  os_info: string
  latency_ms: number
  error: string
}

// ============================
// 系统设置
// ============================

export type ValueType = 'string' | 'number' | 'bool' | 'json'
export type SettingCategory = 'general' | 'notification' | 'auth' | 'deploy'

export interface SystemSetting {
  id: string
  setting_key: string
  value: string
  value_type: ValueType
  category: SettingCategory
  description: string
  updated_by: string
  updated_at: string
}

export interface UpdateSettingPayload {
  setting_key: string
  value: string
  value_type?: ValueType
  category?: SettingCategory
  description?: string
}

export interface BatchUpdateSettingPayload {
  items: UpdateSettingPayload[]
}

// ============================
// 配置
// ============================

export type ConfigScope = 'base' | 'env' | 'customer'

export interface ConfigItem {
  id: string
  service_id: string
  env_id: string
  config_key: string
  config_scope: ConfigScope
  value_type: string
  default_value: string
  encrypted_flag: boolean
  version_no: number
  created_at: string
}

// ============================
// Jenkins
// ============================

export interface JenkinsParamDef {
  name: string
  type: string
  description: string
  default_value: string
  choices?: string[]
}

export interface JenkinsBuildRef {
  number: number
  url: string
}

export interface JenkinsJobInfo {
  description: string
  buildable: boolean
  full_name: string
  url: string
  last_build: JenkinsBuildRef | null
  last_successful_build: JenkinsBuildRef | null
  parameters: JenkinsParamDef[]
}

export interface JenkinsBuildInfo {
  number: number
  result: string
  building: boolean
  timestamp: number
  duration: number
  url: string
}

// ============================
// 容器
// ============================

export type ContainerStatus = 'running' | 'exited' | 'paused' | 'restarting' | 'removed' | 'unknown'

export interface Container {
  id: string
  host_id: string
  container_id: string
  container_name: string
  image: string
  status: ContainerStatus
  config_paths: string[]
  description: string
  last_synced_at: string | null
  created_at: string
  updated_at: string
}

export interface UpdateContainerPayload {
  config_paths?: string[]
  description?: string
}

export interface ContainerConfigFile {
  path: string
  content: string
}

export interface WriteConfigPayload {
  path: string
  content: string
  restart?: boolean
}

export interface ContainerInspect {
  raw: string
}
