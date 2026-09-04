import { http } from '@/lib/http'

export interface BackupTask {
  id: string
  name: string
  backup_type: string
  cron_expr: string
  enabled: boolean
  db_host: string
  db_port: number
  db_user: string
  db_name: string
  target_host_id: string
  target_path: string
  retention_days: number
  description: string
  last_run_at: string | null
  last_run_status: string
  created_at: string
  updated_at: string
}

export interface BackupRecord {
  id: string
  task_id: string
  task_name: string
  status: string
  file_name: string
  file_size: number
  duration: number
  error: string
  started_at: string
  finished_at: string | null
}

export interface MigrationTask {
  id: string
  name: string
  db_type: string
  source_host: string
  source_port: number
  source_user: string
  db_names: string
  target_host: string
  target_port: number
  target_user: string
  mode: string
  description: string
  last_run_at: string | null
  last_run_status: string
  created_at: string
  updated_at: string
}

export interface MigrationRecordItem {
  id: string
  record_id: string
  db_name: string
  action: string
  status: string
  message: string
  started_at: string
  finished_at: string | null
  duration: number
}

export interface MigrationRecord {
  id: string
  task_id: string
  task_name: string
  db_type: string
  mode: string
  status: string
  source_host: string
  target_host: string
  db_names: string
  summary: string
  error: string
  started_at: string
  finished_at: string | null
  duration: number
  items?: MigrationRecordItem[]
}

export interface ObjectSyncTask {
  id: string
  name: string
  source_endpoint: string
  source_region: string
  source_bucket: string
  source_path: string
  source_use_ssl: boolean
  target_endpoint: string
  target_region: string
  target_bucket: string
  target_path: string
  target_use_ssl: boolean
  mode: string
  description: string
  last_run_at: string | null
  last_run_status: string
  created_at: string
  updated_at: string
}

export interface ObjectSyncRecordItem {
  id: string
  record_id: string
  source_key: string
  target_key: string
  size: number
  etag: string
  action: string
  status: string
  message: string
  started_at: string
  finished_at: string | null
  duration: number
}

export interface ObjectSyncRecord {
  id: string
  task_id: string
  task_name: string
  mode: string
  status: string
  source_bucket: string
  source_path: string
  target_bucket: string
  target_path: string
  object_count: number
  success_count: number
  skipped_count: number
  failed_count: number
  bytes_total: number
  summary: string
  error: string
  started_at: string
  finished_at: string | null
  duration: number
  items?: ObjectSyncRecordItem[]
}

export interface CreateBackupTaskPayload {
  name: string
  backup_type: string
  cron_expr: string
  enabled: boolean
  db_host: string
  db_port: number
  db_user: string
  db_password: string
  db_name: string
  target_host_id: string
  target_path: string
  retention_days: number
  description: string
}

export interface UpdateBackupTaskPayload {
  name?: string
  cron_expr?: string
  enabled?: boolean
  db_host?: string
  db_port?: number
  db_user?: string
  db_password?: string
  db_name?: string
  target_host_id?: string
  target_path?: string
  retention_days?: number
  description?: string
}

export interface CreateMigrationTaskPayload {
  name: string
  db_type: string
  source_host: string
  source_port: number
  source_user: string
  source_password: string
  db_names: string
  target_host: string
  target_port: number
  target_user: string
  target_password: string
  mode: string
  description: string
}

export interface UpdateMigrationTaskPayload {
  name?: string
  db_type?: string
  source_host?: string
  source_port?: number
  source_user?: string
  source_password?: string
  db_names?: string
  target_host?: string
  target_port?: number
  target_user?: string
  target_password?: string
  mode?: string
  description?: string
}

export interface CreateObjectSyncTaskPayload {
  name: string
  source_endpoint: string
  source_region: string
  source_bucket: string
  source_path: string
  source_access_key: string
  source_secret_key: string
  source_use_ssl: boolean
  target_endpoint: string
  target_region: string
  target_bucket: string
  target_path: string
  target_access_key: string
  target_secret_key: string
  target_use_ssl: boolean
  mode: string
  description: string
}

export interface UpdateObjectSyncTaskPayload {
  name?: string
  source_endpoint?: string
  source_region?: string
  source_bucket?: string
  source_path?: string
  source_access_key?: string
  source_secret_key?: string
  source_use_ssl?: boolean
  target_endpoint?: string
  target_region?: string
  target_bucket?: string
  target_path?: string
  target_access_key?: string
  target_secret_key?: string
  target_use_ssl?: boolean
  mode?: string
  description?: string
}

export async function listBackupTasks(params?: { keyword?: string; page?: number; page_size?: number }) {
  const res = await http.get<{ list: BackupTask[]; total: number }>('/api/v1/backup/tasks', params)
  return res.data
}

export async function getBackupTask(id: string) {
  const res = await http.get<BackupTask>(`/api/v1/backup/tasks/${id}`)
  return res.data
}

export async function createBackupTask(payload: CreateBackupTaskPayload) {
  const res = await http.post<BackupTask>('/api/v1/backup/tasks', payload)
  return res.data
}

export async function updateBackupTask(id: string, payload: UpdateBackupTaskPayload) {
  const res = await http.put<BackupTask>(`/api/v1/backup/tasks/${id}`, payload)
  return res.data
}

export async function deleteBackupTask(id: string) {
  await http.del(`/api/v1/backup/tasks/${id}`)
}

export async function triggerBackupTask(id: string) {
  await http.post(`/api/v1/backup/tasks/${id}/trigger`)
}

export async function listBackupRecords(params?: { task_id?: string; status?: string; page?: number; page_size?: number }) {
  const res = await http.get<{ list: BackupRecord[]; total: number }>('/api/v1/backup/records', params)
  return res.data
}

export async function listMigrationTasks(params?: { keyword?: string; page?: number; page_size?: number }) {
  const res = await http.get<{ list: MigrationTask[]; total: number }>('/api/v1/backup/migrations', params)
  return res.data
}

export async function createMigrationTask(payload: CreateMigrationTaskPayload) {
  const res = await http.post<MigrationTask>('/api/v1/backup/migrations', payload)
  return res.data
}

export async function updateMigrationTask(id: string, payload: UpdateMigrationTaskPayload) {
  const res = await http.put<MigrationTask>(`/api/v1/backup/migrations/${id}`, payload)
  return res.data
}

export async function deleteMigrationTask(id: string) {
  await http.del(`/api/v1/backup/migrations/${id}`)
}

export async function executeMigrationTask(id: string, payload?: { confirm_overwrite?: boolean }) {
  await http.post(`/api/v1/backup/migrations/${id}/execute`, payload || {})
}

export async function listMigrationRecords(params?: { task_id?: string; status?: string; page?: number; page_size?: number }) {
  const res = await http.get<{ list: MigrationRecord[]; total: number }>('/api/v1/backup/migration-records', params)
  return res.data
}

export async function getMigrationRecord(id: string) {
  const res = await http.get<MigrationRecord>(`/api/v1/backup/migration-records/${id}`)
  return res.data
}

export async function listObjectSyncTasks(params?: { keyword?: string; page?: number; page_size?: number }) {
  const res = await http.get<{ list: ObjectSyncTask[]; total: number }>('/api/v1/backup/object-sync/tasks', params)
  return res.data
}

export async function createObjectSyncTask(payload: CreateObjectSyncTaskPayload) {
  const res = await http.post<ObjectSyncTask>('/api/v1/backup/object-sync/tasks', payload)
  return res.data
}

export async function updateObjectSyncTask(id: string, payload: UpdateObjectSyncTaskPayload) {
  const res = await http.put<ObjectSyncTask>(`/api/v1/backup/object-sync/tasks/${id}`, payload)
  return res.data
}

export async function deleteObjectSyncTask(id: string) {
  await http.del(`/api/v1/backup/object-sync/tasks/${id}`)
}

export async function executeObjectSyncTask(id: string, payload?: { confirm_overwrite?: boolean }) {
  await http.post(`/api/v1/backup/object-sync/tasks/${id}/execute`, payload || {})
}

export async function listObjectSyncRecords(params?: { task_id?: string; status?: string; page?: number; page_size?: number }) {
  const res = await http.get<{ list: ObjectSyncRecord[]; total: number }>('/api/v1/backup/object-sync/records', params)
  return res.data
}

export async function getObjectSyncRecord(id: string) {
  const res = await http.get<ObjectSyncRecord>(`/api/v1/backup/object-sync/records/${id}`)
  return res.data
}
