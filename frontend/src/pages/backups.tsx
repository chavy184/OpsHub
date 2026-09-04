import { useEffect, useState, useCallback } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Plus, Trash2, Play, Database, Pencil } from 'lucide-react'
import { toast } from 'sonner'
import {
  listBackupTasks,
  createBackupTask,
  updateBackupTask,
  deleteBackupTask,
  triggerBackupTask,
  listBackupRecords,
  listMigrationTasks,
  createMigrationTask,
  updateMigrationTask,
  deleteMigrationTask,
  executeMigrationTask,
  listMigrationRecords,
  listObjectSyncTasks,
  createObjectSyncTask,
  updateObjectSyncTask,
  deleteObjectSyncTask,
  executeObjectSyncTask,
  listObjectSyncRecords,
  type BackupTask,
  type BackupRecord,
  type CreateBackupTaskPayload,
  type MigrationTask,
  type MigrationRecord,
  type CreateMigrationTaskPayload,
  type ObjectSyncTask,
  type ObjectSyncRecord,
  type CreateObjectSyncTaskPayload,
} from '@/api/backups'
import { listHosts } from '@/api/hosts'

const STATUS_MAP: Record<string, { label: string; variant: 'default' | 'secondary' | 'outline' | 'destructive' }> = {
  pending: { label: '待执行', variant: 'secondary' },
  running: { label: '执行中', variant: 'default' },
  success: { label: '成功', variant: 'outline' },
  partial_success: { label: '部分成功', variant: 'secondary' },
  skipped: { label: '已跳过', variant: 'secondary' },
  failed: { label: '失败', variant: 'destructive' },
}

const BACKUP_TYPE_OPTIONS = [
  { value: 'postgres', label: 'PostgreSQL' },
  { value: 'mysql', label: 'MySQL' },
]

const BACKUP_SCOPE_OPTIONS = [
  { value: 'full', label: '全量备份' },
  { value: 'selected', label: '选择数据库列表备份' },
]

const MIGRATION_MODE_OPTIONS = [
  { value: 'create_if_missing', label: '不存在则创建，存在则跳过' },
  { value: 'overwrite', label: '覆盖：存在则删除重建' },
]

const OBJECT_SYNC_MODE_OPTIONS = [
  { value: 'copy_if_missing', label: '仅复制目标不存在的对象' },
  { value: 'overwrite', label: '覆盖：目标存在时覆盖写入' },
  { value: 'checksum_skip', label: '校验一致跳过，不一致覆盖' },
]

const SSL_OPTIONS = [
  { value: 'true', label: 'HTTPS' },
  { value: 'false', label: 'HTTP' },
]

type BackupTab = 'tasks' | 'records' | 'migrations' | 'migration-records' | 'object-sync' | 'object-sync-records'
type BackupFeature = 'backup' | 'migration' | 'object-sync'

const BACKUP_TABS: Array<{ value: BackupTab; label: string; path: string; description: string; feature: BackupFeature; mode: 'task' | 'record' }> = [
  { value: 'tasks', label: '任务', path: '/backups/tasks', description: '管理数据库自动备份任务，支持定时执行与手动触发', feature: 'backup', mode: 'task' },
  { value: 'records', label: '记录', path: '/backups/records', description: '查看数据库备份任务的历史执行结果', feature: 'backup', mode: 'record' },
  { value: 'migrations', label: '任务', path: '/backups/migrations', description: '管理数据库数据迁移任务，支持手动执行', feature: 'migration', mode: 'task' },
  { value: 'migration-records', label: '记录', path: '/backups/migration-records', description: '查看数据库迁移任务的历史执行结果', feature: 'migration', mode: 'record' },
  { value: 'object-sync', label: '任务', path: '/backups/object-sync', description: '管理 MinIO / OSS 对象存储同步任务', feature: 'object-sync', mode: 'task' },
  { value: 'object-sync-records', label: '记录', path: '/backups/object-sync-records', description: '查看对象存储同步任务的历史执行结果', feature: 'object-sync', mode: 'record' },
]

const BACKUP_FEATURES: Array<{ value: BackupFeature; label: string; path: string }> = [
  { value: 'backup', label: '数据库备份', path: '/backups/tasks' },
  { value: 'migration', label: '数据库迁移', path: '/backups/migrations' },
  { value: 'object-sync', label: '对象同步', path: '/backups/object-sync' },
]

function tabFromPath(pathname: string): BackupTab {
  return BACKUP_TABS.find((item) => item.path === pathname)?.value || 'tasks'
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
}

export default function BackupsPage() {
  const location = useLocation()
  const navigate = useNavigate()
  const tab = tabFromPath(location.pathname)
  const currentTab = BACKUP_TABS.find((item) => item.value === tab) || BACKUP_TABS[0]!
  const currentFeature = currentTab.feature
  const currentFeatureMeta = BACKUP_FEATURES.find((item) => item.value === currentFeature) || BACKUP_FEATURES[0]!
  const currentFeatureTabs = BACKUP_TABS.filter((item) => item.feature === currentFeature)
  const [tasks, setTasks] = useState<BackupTask[]>([])
  const [records, setRecords] = useState<BackupRecord[]>([])
  const [migrationTasks, setMigrationTasks] = useState<MigrationTask[]>([])
  const [migrationRecords, setMigrationRecords] = useState<MigrationRecord[]>([])
  const [objectSyncTasks, setObjectSyncTasks] = useState<ObjectSyncTask[]>([])
  const [objectSyncRecords, setObjectSyncRecords] = useState<ObjectSyncRecord[]>([])
  const [hosts, setHosts] = useState<{ id: string; name: string; host_address: string }[]>([])
  const [loading, setLoading] = useState(true)
  const [migrationLoading, setMigrationLoading] = useState(true)
  const [objectSyncLoading, setObjectSyncLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [showMigrationCreate, setShowMigrationCreate] = useState(false)
  const [showObjectSyncCreate, setShowObjectSyncCreate] = useState(false)
  const [editTarget, setEditTarget] = useState<BackupTask | null>(null)
  const [editMigrationTarget, setEditMigrationTarget] = useState<MigrationTask | null>(null)
  const [editObjectSyncTarget, setEditObjectSyncTarget] = useState<ObjectSyncTask | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)
  const [deleteMigrationTarget, setDeleteMigrationTarget] = useState<string | null>(null)
  const [deleteObjectSyncTarget, setDeleteObjectSyncTarget] = useState<string | null>(null)
  const [executeMigrationTarget, setExecuteMigrationTarget] = useState<MigrationTask | null>(null)
  const [executeObjectSyncTarget, setExecuteObjectSyncTarget] = useState<ObjectSyncTask | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [backupScope, setBackupScope] = useState<'full' | 'selected'>('full')

  const defaultForm: CreateBackupTaskPayload = {
    name: '',
    backup_type: 'postgres',
    cron_expr: '0 2 * * *',
    enabled: true,
    db_host: '',
    db_port: 5432,
    db_user: 'postgres',
    db_password: '',
    db_name: '',
    target_host_id: '',
    target_path: '/data/backups',
    retention_days: 10,
    description: '',
  }
  const [form, setForm] = useState<CreateBackupTaskPayload>(defaultForm)

  const defaultMigrationForm: CreateMigrationTaskPayload = {
    name: '',
    db_type: 'postgres',
    source_host: '',
    source_port: 5432,
    source_user: 'postgres',
    source_password: '',
    db_names: '',
    target_host: '',
    target_port: 5432,
    target_user: 'postgres',
    target_password: '',
    mode: 'create_if_missing',
    description: '',
  }
  const [migrationForm, setMigrationForm] = useState<CreateMigrationTaskPayload>(defaultMigrationForm)

  const defaultObjectSyncForm: CreateObjectSyncTaskPayload = {
    name: '',
    source_endpoint: '',
    source_region: 'us-east-1',
    source_bucket: '',
    source_path: '',
    source_access_key: '',
    source_secret_key: '',
    source_use_ssl: true,
    target_endpoint: '',
    target_region: 'us-east-1',
    target_bucket: '',
    target_path: '',
    target_access_key: '',
    target_secret_key: '',
    target_use_ssl: true,
    mode: 'copy_if_missing',
    description: '',
  }
  const [objectSyncForm, setObjectSyncForm] = useState<CreateObjectSyncTaskPayload>(defaultObjectSyncForm)

  const normalizeDBList = (value: string) => value
    .split(/[\n,]/)
    .map((name) => name.trim())
    .filter(Boolean)
    .join(',')

  const formatBackupScope = (dbName: string) => {
    if (!dbName?.trim()) return '全量备份'
    const dbNames = dbName.split(/[\n,]/).map((name) => name.trim()).filter(Boolean)
    return dbNames.length > 1 ? `指定 ${dbNames.length} 个库` : dbNames[0]
  }

  const buildSubmitPayload = () => ({
    ...form,
    db_name: backupScope === 'full' ? '' : normalizeDBList(form.db_name),
  })

  const buildMigrationSubmitPayload = () => ({
    ...migrationForm,
    db_names: normalizeDBList(migrationForm.db_names),
  })

  const openCreate = () => {
    setForm(defaultForm)
    setBackupScope('full')
    setShowCreate(true)
  }

  const openMigrationCreate = () => {
    setMigrationForm(defaultMigrationForm)
    setShowMigrationCreate(true)
  }

  const openObjectSyncCreate = () => {
    setObjectSyncForm(defaultObjectSyncForm)
    setShowObjectSyncCreate(true)
  }

  const fetchTasks = useCallback(async () => {
    setLoading(true)
    try {
      const data = await listBackupTasks({ page: 1, page_size: 50 })
      setTasks(data.list || [])
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchHosts = useCallback(async () => {
    try {
      const res = await listHosts({ page: 1, page_size: 100 })
      setHosts((res as any).list || res || [])
    } catch { /* ignore */ }
  }, [])

  const fetchRecords = useCallback(async (taskId?: string) => {
    try {
      const data = await listBackupRecords({ task_id: taskId, page: 1, page_size: 50 })
      setRecords(data.list || [])
    } catch { /* ignore */ }
  }, [])

  const fetchMigrationTasks = useCallback(async () => {
    setMigrationLoading(true)
    try {
      const data = await listMigrationTasks({ page: 1, page_size: 50 })
      setMigrationTasks(data.list || [])
    } finally {
      setMigrationLoading(false)
    }
  }, [])

  const fetchMigrationRecords = useCallback(async (taskId?: string) => {
    try {
      const data = await listMigrationRecords({ task_id: taskId, page: 1, page_size: 50 })
      setMigrationRecords(data.list || [])
    } catch { /* ignore */ }
  }, [])

  const fetchObjectSyncTasks = useCallback(async () => {
    setObjectSyncLoading(true)
    try {
      const data = await listObjectSyncTasks({ page: 1, page_size: 50 })
      setObjectSyncTasks(data.list || [])
    } finally {
      setObjectSyncLoading(false)
    }
  }, [])

  const fetchObjectSyncRecords = useCallback(async (taskId?: string) => {
    try {
      const data = await listObjectSyncRecords({ task_id: taskId, page: 1, page_size: 50 })
      setObjectSyncRecords(data.list || [])
    } catch { /* ignore */ }
  }, [])

  useEffect(() => {
    fetchTasks()
    fetchHosts()
  }, [fetchTasks, fetchHosts])

  useEffect(() => {
    if (tab === 'records') fetchRecords()
    if (tab === 'migrations') fetchMigrationTasks()
    if (tab === 'migration-records') fetchMigrationRecords()
    if (tab === 'object-sync') fetchObjectSyncTasks()
    if (tab === 'object-sync-records') fetchObjectSyncRecords()
  }, [tab, fetchRecords, fetchMigrationTasks, fetchMigrationRecords, fetchObjectSyncTasks, fetchObjectSyncRecords])

  const handleCreate = async () => {
    if (!form.name.trim() || !form.db_host.trim() || !form.target_host_id) {
      toast.error('请填写必要字段')
      return
    }
    if (backupScope === 'selected' && !normalizeDBList(form.db_name)) {
      toast.error('请填写要备份的数据库列表')
      return
    }
    setSubmitting(true)
    try {
      await createBackupTask(buildSubmitPayload())
      toast.success('备份任务已创建')
      setShowCreate(false)
      setForm(defaultForm)
      setBackupScope('full')
      fetchTasks()
    } catch (err: any) {
      toast.error(err?.message || '创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleUpdate = async () => {
    if (!editTarget) return
    if (backupScope === 'selected' && !normalizeDBList(form.db_name)) {
      toast.error('请填写要备份的数据库列表')
      return
    }
    setSubmitting(true)
    try {
      await updateBackupTask(editTarget.id, buildSubmitPayload())
      toast.success('已更新')
      setEditTarget(null)
      fetchTasks()
    } catch (err: any) {
      toast.error(err?.message || '更新失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setSubmitting(true)
    try {
      await deleteBackupTask(deleteTarget)
      toast.success('已删除')
      setDeleteTarget(null)
      fetchTasks()
    } catch (err: any) {
      toast.error(err?.message || '删除失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleCreateMigration = async () => {
    if (!migrationForm.name.trim() || !migrationForm.source_host.trim() || !migrationForm.target_host.trim() || !normalizeDBList(migrationForm.db_names)) {
      toast.error('请填写迁移任务必要字段')
      return
    }
    setSubmitting(true)
    try {
      await createMigrationTask(buildMigrationSubmitPayload())
      toast.success('迁移任务已创建')
      setShowMigrationCreate(false)
      setMigrationForm(defaultMigrationForm)
      fetchMigrationTasks()
    } catch (err: any) {
      toast.error(err?.message || '创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleUpdateMigration = async () => {
    if (!editMigrationTarget) return
    if (!normalizeDBList(migrationForm.db_names)) {
      toast.error('请填写数据库列表')
      return
    }
    setSubmitting(true)
    try {
      await updateMigrationTask(editMigrationTarget.id, buildMigrationSubmitPayload())
      toast.success('已更新')
      setEditMigrationTarget(null)
      fetchMigrationTasks()
    } catch (err: any) {
      toast.error(err?.message || '更新失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDeleteMigration = async () => {
    if (!deleteMigrationTarget) return
    setSubmitting(true)
    try {
      await deleteMigrationTask(deleteMigrationTarget)
      toast.success('已删除')
      setDeleteMigrationTarget(null)
      fetchMigrationTasks()
    } catch (err: any) {
      toast.error(err?.message || '删除失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleExecuteMigration = async () => {
    if (!executeMigrationTarget) return
    setSubmitting(true)
    try {
      await executeMigrationTask(executeMigrationTarget.id, { confirm_overwrite: executeMigrationTarget.mode === 'overwrite' })
      toast.success('迁移任务已触发，后台执行中')
      setExecuteMigrationTarget(null)
      fetchMigrationRecords()
    } catch (err: any) {
      toast.error(err?.message || '执行失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleCreateObjectSync = async () => {
    if (!objectSyncForm.name.trim() || !objectSyncForm.source_endpoint.trim() || !objectSyncForm.source_bucket.trim() || !objectSyncForm.target_endpoint.trim() || !objectSyncForm.target_bucket.trim()) {
      toast.error('请填写对象同步任务必要字段')
      return
    }
    setSubmitting(true)
    try {
      await createObjectSyncTask(objectSyncForm)
      toast.success('对象同步任务已创建')
      setShowObjectSyncCreate(false)
      setObjectSyncForm(defaultObjectSyncForm)
      fetchObjectSyncTasks()
    } catch (err: any) {
      toast.error(err?.message || '创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleUpdateObjectSync = async () => {
    if (!editObjectSyncTarget) return
    setSubmitting(true)
    try {
      await updateObjectSyncTask(editObjectSyncTarget.id, objectSyncForm)
      toast.success('已更新')
      setEditObjectSyncTarget(null)
      fetchObjectSyncTasks()
    } catch (err: any) {
      toast.error(err?.message || '更新失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDeleteObjectSync = async () => {
    if (!deleteObjectSyncTarget) return
    setSubmitting(true)
    try {
      await deleteObjectSyncTask(deleteObjectSyncTarget)
      toast.success('已删除')
      setDeleteObjectSyncTarget(null)
      fetchObjectSyncTasks()
    } catch (err: any) {
      toast.error(err?.message || '删除失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleExecuteObjectSync = async () => {
    if (!executeObjectSyncTarget) return
    setSubmitting(true)
    try {
      await executeObjectSyncTask(executeObjectSyncTarget.id, { confirm_overwrite: executeObjectSyncTarget.mode === 'overwrite' })
      toast.success('对象同步任务已触发，后台执行中')
      setExecuteObjectSyncTarget(null)
      fetchObjectSyncRecords()
    } catch (err: any) {
      toast.error(err?.message || '执行失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleTrigger = async (id: string) => {
    try {
      await triggerBackupTask(id)
      toast.success('备份任务已触发，后台执行中')
    } catch (err: any) {
      toast.error(err?.message || '触发失败')
    }
  }

  const handleToggleEnabled = async (task: BackupTask) => {
    try {
      await updateBackupTask(task.id, { enabled: !task.enabled })
      fetchTasks()
    } catch (err: any) {
      toast.error(err?.message || '切换失败')
    }
  }

  const openEdit = (task: BackupTask) => {
    setForm({
      name: task.name,
      backup_type: task.backup_type,
      cron_expr: task.cron_expr,
      enabled: task.enabled,
      db_host: task.db_host,
      db_port: task.db_port,
      db_user: task.db_user,
      db_password: '',
      db_name: task.db_name,
      target_host_id: task.target_host_id,
      target_path: task.target_path,
      retention_days: task.retention_days,
      description: task.description,
    })
    setBackupScope(task.db_name?.trim() ? 'selected' : 'full')
    setEditTarget(task)
  }

  const openMigrationEdit = (task: MigrationTask) => {
    setMigrationForm({
      name: task.name,
      db_type: task.db_type,
      source_host: task.source_host,
      source_port: task.source_port,
      source_user: task.source_user,
      source_password: '',
      db_names: task.db_names,
      target_host: task.target_host,
      target_port: task.target_port,
      target_user: task.target_user,
      target_password: '',
      mode: task.mode,
      description: task.description,
    })
    setEditMigrationTarget(task)
  }

  const openObjectSyncEdit = (task: ObjectSyncTask) => {
    setObjectSyncForm({
      name: task.name,
      source_endpoint: task.source_endpoint,
      source_region: task.source_region || 'us-east-1',
      source_bucket: task.source_bucket,
      source_path: task.source_path,
      source_access_key: '',
      source_secret_key: '',
      source_use_ssl: task.source_use_ssl,
      target_endpoint: task.target_endpoint,
      target_region: task.target_region || 'us-east-1',
      target_bucket: task.target_bucket,
      target_path: task.target_path,
      target_access_key: '',
      target_secret_key: '',
      target_use_ssl: task.target_use_ssl,
      mode: task.mode,
      description: task.description,
    })
    setEditObjectSyncTarget(task)
  }

  const migrationModeLabel = (mode: string) => MIGRATION_MODE_OPTIONS.find((opt) => opt.value === mode)?.label || mode
  const objectSyncModeLabel = (mode: string) => OBJECT_SYNC_MODE_OPTIONS.find((opt) => opt.value === mode)?.label || mode

  const hostName = (id: string) => {
    const h = hosts.find((h) => h.id === id)
    return h ? `${h.name} (${h.host_address})` : id.slice(0, 8)
  }

  // ============================
  // Render
  // ============================

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">{currentFeatureMeta.label}</h1>
          <p className="text-sm text-muted-foreground mt-1">{currentTab.description}</p>
        </div>
        <div className="flex gap-2">
          {tab === 'tasks' && (
            <Button size="sm" onClick={openCreate}>
              <Plus className="w-4 h-4 mr-1" /> 新建任务
            </Button>
          )}
          {tab === 'migrations' && (
            <Button size="sm" onClick={openMigrationCreate}>
              <Plus className="w-4 h-4 mr-1" /> 新建迁移
            </Button>
          )}
          {tab === 'object-sync' && (
            <Button size="sm" onClick={openObjectSyncCreate}>
              <Plus className="w-4 h-4 mr-1" /> 新建同步
            </Button>
          )}
        </div>
      </div>

      <div className="flex items-center justify-between gap-4 border-b pb-3">
        <div className="flex gap-2">
          {BACKUP_FEATURES.map((item) => (
            <Button
              key={item.value}
              variant={currentFeature === item.value ? 'default' : 'outline'}
              size="sm"
              onClick={() => navigate(item.path)}
            >
              {item.label}
            </Button>
          ))}
        </div>
        <div className="flex gap-1 rounded-md border bg-muted/30 p-1">
          {currentFeatureTabs.map((item) => (
            <Button
              key={item.value}
              variant={tab === item.value ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => navigate(item.path)}
            >
              {item.label}
            </Button>
          ))}
        </div>
      </div>

      {/* Tasks Tab */}
      {tab === 'tasks' && (
        loading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => <Skeleton key={i} className="h-12 w-full" />)}
          </div>
        ) : tasks.length === 0 ? (
          <div className="text-center py-16 text-muted-foreground">
            <Database className="w-12 h-12 mx-auto mb-4 opacity-30" />
            <p>暂无备份任务</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>任务名称</TableHead>
                <TableHead>类型</TableHead>
                <TableHead>备份范围</TableHead>
                <TableHead>调度</TableHead>
                <TableHead>目标主机</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>上次执行</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tasks.map((task) => (
                <TableRow key={task.id}>
                  <TableCell className="font-medium">{task.name}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{task.backup_type === 'postgres' ? 'PostgreSQL' : 'MySQL'}</Badge>
                  </TableCell>
                  <TableCell className="text-xs">{formatBackupScope(task.db_name)}</TableCell>
                  <TableCell className="font-mono text-xs">{task.cron_expr}</TableCell>
                  <TableCell className="text-xs">{hostName(task.target_host_id)}</TableCell>
                  <TableCell>
                    <Badge
                      variant={task.enabled ? 'default' : 'secondary'}
                      className="cursor-pointer"
                      onClick={() => handleToggleEnabled(task)}
                    >
                      {task.enabled ? '已启用' : '已禁用'}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs">
                    {task.last_run_at ? (
                      <span className="flex items-center gap-1">
                        {task.last_run_at}
                        {task.last_run_status && (
                          <Badge variant={STATUS_MAP[task.last_run_status]?.variant || 'outline'} className="text-[10px] px-1">
                            {STATUS_MAP[task.last_run_status]?.label || task.last_run_status}
                          </Badge>
                        )}
                      </span>
                    ) : (
                      <span className="text-muted-foreground">从未执行</span>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button variant="ghost" size="icon" title="立即执行" onClick={() => handleTrigger(task.id)}>
                        <Play className="w-4 h-4" />
                      </Button>
                      <Button variant="ghost" size="icon" title="编辑" onClick={() => openEdit(task)}>
                        <Pencil className="w-4 h-4" />
                      </Button>
                      <Button variant="ghost" size="icon" title="删除" onClick={() => setDeleteTarget(task.id)}>
                        <Trash2 className="w-4 h-4 text-destructive" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )
      )}

      {/* Records Tab */}
      {tab === 'records' && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>任务名称</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>文件名</TableHead>
              <TableHead>大小</TableHead>
              <TableHead>耗时</TableHead>
              <TableHead>开始时间</TableHead>
              <TableHead>错误</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {records.length === 0 ? (
              <TableRow><TableCell colSpan={7} className="text-center text-muted-foreground py-8">暂无记录</TableCell></TableRow>
            ) : records.map((r) => (
              <TableRow key={r.id}>
                <TableCell>{r.task_name}</TableCell>
                <TableCell>
                  <Badge variant={STATUS_MAP[r.status]?.variant || 'outline'}>
                    {STATUS_MAP[r.status]?.label || r.status}
                  </Badge>
                </TableCell>
                <TableCell className="font-mono text-xs max-w-[200px] truncate">{r.file_name || '-'}</TableCell>
                <TableCell className="text-xs">{r.file_size ? formatFileSize(r.file_size) : '-'}</TableCell>
                <TableCell className="text-xs">{r.duration ? `${r.duration}s` : '-'}</TableCell>
                <TableCell className="text-xs">{r.started_at}</TableCell>
                <TableCell className="text-xs text-destructive max-w-[200px] truncate" title={r.error}>{r.error || '-'}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {/* Migration Tasks Tab */}
      {tab === 'migrations' && (
        migrationLoading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => <Skeleton key={i} className="h-12 w-full" />)}
          </div>
        ) : migrationTasks.length === 0 ? (
          <div className="text-center py-16 text-muted-foreground">
            <Database className="w-12 h-12 mx-auto mb-4 opacity-30" />
            <p>暂无迁移任务</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>任务名称</TableHead>
                <TableHead>类型</TableHead>
                <TableHead>源地址</TableHead>
                <TableHead>目标地址</TableHead>
                <TableHead>数据库</TableHead>
                <TableHead>模式</TableHead>
                <TableHead>上次执行</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {migrationTasks.map((task) => (
                <TableRow key={task.id}>
                  <TableCell className="font-medium">{task.name}</TableCell>
                  <TableCell><Badge variant="outline">{task.db_type === 'postgres' ? 'PostgreSQL' : 'MySQL'}</Badge></TableCell>
                  <TableCell className="text-xs">{task.source_host}:{task.source_port}</TableCell>
                  <TableCell className="text-xs">{task.target_host}:{task.target_port}</TableCell>
                  <TableCell className="text-xs">{formatBackupScope(task.db_names)}</TableCell>
                  <TableCell className="text-xs">{migrationModeLabel(task.mode)}</TableCell>
                  <TableCell className="text-xs">
                    {task.last_run_at ? (
                      <span className="flex items-center gap-1">
                        {task.last_run_at}
                        {task.last_run_status && (
                          <Badge variant={STATUS_MAP[task.last_run_status]?.variant || 'outline'} className="text-[10px] px-1">
                            {STATUS_MAP[task.last_run_status]?.label || task.last_run_status}
                          </Badge>
                        )}
                      </span>
                    ) : (
                      <span className="text-muted-foreground">从未执行</span>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button variant="ghost" size="icon" title="执行迁移" onClick={() => setExecuteMigrationTarget(task)}>
                        <Play className="w-4 h-4" />
                      </Button>
                      <Button variant="ghost" size="icon" title="编辑" onClick={() => openMigrationEdit(task)}>
                        <Pencil className="w-4 h-4" />
                      </Button>
                      <Button variant="ghost" size="icon" title="删除" onClick={() => setDeleteMigrationTarget(task.id)}>
                        <Trash2 className="w-4 h-4 text-destructive" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )
      )}

      {/* Migration Records Tab */}
      {tab === 'migration-records' && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>任务名称</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>源地址</TableHead>
              <TableHead>目标地址</TableHead>
              <TableHead>摘要</TableHead>
              <TableHead>耗时</TableHead>
              <TableHead>开始时间</TableHead>
              <TableHead>错误</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {migrationRecords.length === 0 ? (
              <TableRow><TableCell colSpan={9} className="text-center text-muted-foreground py-8">暂无迁移记录</TableCell></TableRow>
            ) : migrationRecords.map((r) => (
              <TableRow key={r.id}>
                <TableCell>{r.task_name}</TableCell>
                <TableCell>
                  <Badge variant={STATUS_MAP[r.status]?.variant || (r.status === 'partial_success' ? 'secondary' : 'outline')}>
                    {STATUS_MAP[r.status]?.label || (r.status === 'partial_success' ? '部分成功' : r.status)}
                  </Badge>
                </TableCell>
                <TableCell className="text-xs">{r.db_type === 'postgres' ? 'PostgreSQL' : 'MySQL'}</TableCell>
                <TableCell className="text-xs">{r.source_host}</TableCell>
                <TableCell className="text-xs">{r.target_host}</TableCell>
                <TableCell className="text-xs">{r.summary || '-'}</TableCell>
                <TableCell className="text-xs">{r.duration ? `${r.duration}s` : '-'}</TableCell>
                <TableCell className="text-xs">{r.started_at}</TableCell>
                <TableCell className="text-xs text-destructive max-w-[200px] truncate" title={r.error}>{r.error || '-'}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {/* Object Sync Tasks Tab */}
      {tab === 'object-sync' && (
        objectSyncLoading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => <Skeleton key={i} className="h-12 w-full" />)}
          </div>
        ) : objectSyncTasks.length === 0 ? (
          <div className="text-center py-16 text-muted-foreground">
            <Database className="w-12 h-12 mx-auto mb-4 opacity-30" />
            <p>暂无对象同步任务</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>任务名称</TableHead>
                <TableHead>源路径</TableHead>
                <TableHead>目标路径</TableHead>
                <TableHead>模式</TableHead>
                <TableHead>上次执行</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {objectSyncTasks.map((task) => (
                <TableRow key={task.id}>
                  <TableCell className="font-medium">{task.name}</TableCell>
                  <TableCell className="font-mono text-xs max-w-[220px] truncate" title={`${task.source_endpoint}/${task.source_bucket}/${task.source_path}`}>
                    {task.source_bucket}/{task.source_path || '-'}
                  </TableCell>
                  <TableCell className="font-mono text-xs max-w-[220px] truncate" title={`${task.target_endpoint}/${task.target_bucket}/${task.target_path}`}>
                    {task.target_bucket}/{task.target_path || '-'}
                  </TableCell>
                  <TableCell className="text-xs">{objectSyncModeLabel(task.mode)}</TableCell>
                  <TableCell className="text-xs">
                    {task.last_run_at ? (
                      <span className="flex items-center gap-1">
                        {task.last_run_at}
                        {task.last_run_status && (
                          <Badge variant={STATUS_MAP[task.last_run_status]?.variant || 'outline'} className="text-[10px] px-1">
                            {STATUS_MAP[task.last_run_status]?.label || task.last_run_status}
                          </Badge>
                        )}
                      </span>
                    ) : (
                      <span className="text-muted-foreground">从未执行</span>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button variant="ghost" size="icon" title="执行同步" onClick={() => setExecuteObjectSyncTarget(task)}>
                        <Play className="w-4 h-4" />
                      </Button>
                      <Button variant="ghost" size="icon" title="编辑" onClick={() => openObjectSyncEdit(task)}>
                        <Pencil className="w-4 h-4" />
                      </Button>
                      <Button variant="ghost" size="icon" title="删除" onClick={() => setDeleteObjectSyncTarget(task.id)}>
                        <Trash2 className="w-4 h-4 text-destructive" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )
      )}

      {/* Object Sync Records Tab */}
      {tab === 'object-sync-records' && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>任务名称</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>源</TableHead>
              <TableHead>目标</TableHead>
              <TableHead>统计</TableHead>
              <TableHead>数据量</TableHead>
              <TableHead>耗时</TableHead>
              <TableHead>开始时间</TableHead>
              <TableHead>错误</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {objectSyncRecords.length === 0 ? (
              <TableRow><TableCell colSpan={9} className="text-center text-muted-foreground py-8">暂无同步记录</TableCell></TableRow>
            ) : objectSyncRecords.map((r) => (
              <TableRow key={r.id}>
                <TableCell>{r.task_name}</TableCell>
                <TableCell>
                  <Badge variant={STATUS_MAP[r.status]?.variant || 'outline'}>
                    {STATUS_MAP[r.status]?.label || r.status}
                  </Badge>
                </TableCell>
                <TableCell className="font-mono text-xs">{r.source_bucket}/{r.source_path || '-'}</TableCell>
                <TableCell className="font-mono text-xs">{r.target_bucket}/{r.target_path || '-'}</TableCell>
                <TableCell className="text-xs">总 {r.object_count} / 成功 {r.success_count} / 跳过 {r.skipped_count} / 失败 {r.failed_count}</TableCell>
                <TableCell className="text-xs">{formatFileSize(r.bytes_total || 0)}</TableCell>
                <TableCell className="text-xs">{r.duration ? `${r.duration}s` : '-'}</TableCell>
                <TableCell className="text-xs">{r.started_at}</TableCell>
                <TableCell className="text-xs text-destructive max-w-[200px] truncate" title={r.error}>{r.error || '-'}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {/* Create / Edit Dialog */}
      <Dialog open={showCreate || !!editTarget} onOpenChange={() => { setShowCreate(false); setEditTarget(null) }}>
        <DialogContent className="max-w-lg max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{editTarget ? '编辑备份任务' : '新建备份任务'}</DialogTitle>
            <DialogDescription>配置数据库连接、调度策略和存储目标</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label>任务名称 *</Label>
              <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="例：生产库每日备份" />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>数据库类型</Label>
                <Select value={form.backup_type} options={BACKUP_TYPE_OPTIONS} onChange={(e) => setForm({ ...form, backup_type: e.target.value, db_port: e.target.value === 'postgres' ? 5432 : 3306 })} />
              </div>
              <div className="grid gap-2">
                <Label>Cron 表达式</Label>
                <Input value={form.cron_expr} onChange={(e) => setForm({ ...form, cron_expr: e.target.value })} placeholder="0 2 * * *" />
                <p className="text-xs text-muted-foreground">标准 5 位 cron，如 "0 2 * * *" = 每日凌晨 2 点</p>
              </div>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div className="grid gap-2">
                <Label>数据库地址 *</Label>
                <Input value={form.db_host} onChange={(e) => setForm({ ...form, db_host: e.target.value })} placeholder="192.168.1.100" />
              </div>
              <div className="grid gap-2">
                <Label>端口</Label>
                <Input type="number" value={form.db_port} onChange={(e) => setForm({ ...form, db_port: Number(e.target.value) })} />
              </div>
              <div className="grid gap-2">
                <Label>用户名 *</Label>
                <Input value={form.db_user} onChange={(e) => setForm({ ...form, db_user: e.target.value })} />
              </div>
            </div>
            <div className="grid gap-2">
              <Label>备份范围</Label>
              <Select
                value={backupScope}
                options={BACKUP_SCOPE_OPTIONS}
                onChange={(e) => {
                  const value = e.target.value as 'full' | 'selected'
                  setBackupScope(value)
                  if (value === 'full') setForm({ ...form, db_name: '' })
                }}
              />
            </div>
            {backupScope === 'selected' && (
              <div className="grid gap-2">
                <Label>数据库列表 *</Label>
                <Textarea
                  value={form.db_name}
                  onChange={(e) => setForm({ ...form, db_name: e.target.value })}
                  placeholder="每行一个数据库名，也可以用英文逗号分隔"
                  rows={3}
                />
              </div>
            )}
            <div className="grid gap-2">
              <div className="grid gap-2">
                <Label>数据库密码 *</Label>
                <Input type="password" value={form.db_password} onChange={(e) => setForm({ ...form, db_password: e.target.value })} placeholder={editTarget ? '留空则不修改' : ''} />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>目标存储主机 *</Label>
                <Select value={form.target_host_id} options={hosts.map((h) => ({ value: h.id, label: `${h.name} (${h.host_address})` }))} placeholder="请选择" onChange={(e) => setForm({ ...form, target_host_id: e.target.value })} />
              </div>
              <div className="grid gap-2">
                <Label>存储路径</Label>
                <Input value={form.target_path} onChange={(e) => setForm({ ...form, target_path: e.target.value })} placeholder="/data/backups" />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>保留天数</Label>
                <Input type="number" value={form.retention_days} onChange={(e) => setForm({ ...form, retention_days: Number(e.target.value) })} />
              </div>
              <div className="grid gap-2">
                <Label>描述</Label>
                <Input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowCreate(false); setEditTarget(null) }}>取消</Button>
            <Button onClick={editTarget ? handleUpdate : handleCreate} disabled={submitting}>
              {submitting ? '提交中...' : (editTarget ? '保存' : '创建')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Migration Create / Edit Dialog */}
      <Dialog open={showMigrationCreate || !!editMigrationTarget} onOpenChange={() => { setShowMigrationCreate(false); setEditMigrationTarget(null) }}>
        <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{editMigrationTarget ? '编辑迁移任务' : '新建迁移任务'}</DialogTitle>
            <DialogDescription>配置源数据库、目标数据库和手动迁移策略</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>任务名称 *</Label>
                <Input value={migrationForm.name} onChange={(e) => setMigrationForm({ ...migrationForm, name: e.target.value })} placeholder="例：生产库同步到测试库" />
              </div>
              <div className="grid gap-2">
                <Label>数据库类型</Label>
                <Select
                  value={migrationForm.db_type}
                  options={BACKUP_TYPE_OPTIONS}
                  onChange={(e) => {
                    const dbType = e.target.value
                    const port = dbType === 'postgres' ? 5432 : 3306
                    setMigrationForm({
                      ...migrationForm,
                      db_type: dbType,
                      source_port: port,
                      target_port: port,
                      source_user: dbType === 'postgres' ? 'postgres' : 'root',
                      target_user: dbType === 'postgres' ? 'postgres' : 'root',
                    })
                  }}
                />
              </div>
            </div>

            <div className="grid gap-3">
              <Label>源数据库</Label>
              <div className="grid grid-cols-4 gap-3">
                <Input value={migrationForm.source_host} onChange={(e) => setMigrationForm({ ...migrationForm, source_host: e.target.value })} placeholder="IP 地址" />
                <Input type="number" value={migrationForm.source_port} onChange={(e) => setMigrationForm({ ...migrationForm, source_port: Number(e.target.value) })} />
                <Input value={migrationForm.source_user} onChange={(e) => setMigrationForm({ ...migrationForm, source_user: e.target.value })} placeholder="用户名" />
                <Input type="password" value={migrationForm.source_password} onChange={(e) => setMigrationForm({ ...migrationForm, source_password: e.target.value })} placeholder={editMigrationTarget ? '留空则不修改' : '密码'} />
              </div>
            </div>

            <div className="grid gap-2">
              <Label>数据库列表 *</Label>
              <Textarea
                value={migrationForm.db_names}
                onChange={(e) => setMigrationForm({ ...migrationForm, db_names: e.target.value })}
                placeholder="每行一个数据库名，也可以用英文逗号分隔"
                rows={3}
              />
            </div>

            <div className="grid gap-3">
              <Label>目标数据库</Label>
              <div className="grid grid-cols-4 gap-3">
                <Input value={migrationForm.target_host} onChange={(e) => setMigrationForm({ ...migrationForm, target_host: e.target.value })} placeholder="IP 地址" />
                <Input type="number" value={migrationForm.target_port} onChange={(e) => setMigrationForm({ ...migrationForm, target_port: Number(e.target.value) })} />
                <Input value={migrationForm.target_user} onChange={(e) => setMigrationForm({ ...migrationForm, target_user: e.target.value })} placeholder="用户名" />
                <Input type="password" value={migrationForm.target_password} onChange={(e) => setMigrationForm({ ...migrationForm, target_password: e.target.value })} placeholder={editMigrationTarget ? '留空则不修改' : '密码'} />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>迁移模式</Label>
                <Select value={migrationForm.mode} options={MIGRATION_MODE_OPTIONS} onChange={(e) => setMigrationForm({ ...migrationForm, mode: e.target.value })} />
              </div>
              <div className="grid gap-2">
                <Label>描述</Label>
                <Input value={migrationForm.description} onChange={(e) => setMigrationForm({ ...migrationForm, description: e.target.value })} />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowMigrationCreate(false); setEditMigrationTarget(null) }}>取消</Button>
            <Button onClick={editMigrationTarget ? handleUpdateMigration : handleCreateMigration} disabled={submitting}>
              {submitting ? '提交中...' : (editMigrationTarget ? '保存' : '创建')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Object Sync Create / Edit Dialog */}
      <Dialog open={showObjectSyncCreate || !!editObjectSyncTarget} onOpenChange={() => { setShowObjectSyncCreate(false); setEditObjectSyncTarget(null) }}>
        <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{editObjectSyncTarget ? '编辑对象同步任务' : '新建对象同步任务'}</DialogTitle>
            <DialogDescription>配置源对象存储、目标对象存储和手动同步策略</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label>任务名称 *</Label>
                <Input value={objectSyncForm.name} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, name: e.target.value })} placeholder="例：MinIO 生产数据同步" />
              </div>
              <div className="grid gap-2">
                <Label>同步模式</Label>
                <Select value={objectSyncForm.mode} options={OBJECT_SYNC_MODE_OPTIONS} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, mode: e.target.value })} />
              </div>
            </div>

            <div className="grid gap-3">
              <Label>源对象存储</Label>
              <div className="grid grid-cols-3 gap-3">
                <Input value={objectSyncForm.source_endpoint} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, source_endpoint: e.target.value })} placeholder="IP:端口 或 URL" />
                <Input value={objectSyncForm.source_region} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, source_region: e.target.value })} placeholder="region" />
                <Select value={String(objectSyncForm.source_use_ssl)} options={SSL_OPTIONS} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, source_use_ssl: e.target.value === 'true' })} />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <Input value={objectSyncForm.source_bucket} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, source_bucket: e.target.value })} placeholder="Bucket *" />
                <Input value={objectSyncForm.source_path} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, source_path: e.target.value })} placeholder="路径/前缀，可为空" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <Input value={objectSyncForm.source_access_key} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, source_access_key: e.target.value })} placeholder={editObjectSyncTarget ? 'Access Key 留空则不修改' : 'Access Key *'} />
                <Input type="password" value={objectSyncForm.source_secret_key} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, source_secret_key: e.target.value })} placeholder={editObjectSyncTarget ? 'Secret Key 留空则不修改' : 'Secret Key *'} />
              </div>
            </div>

            <div className="grid gap-3">
              <Label>目标对象存储</Label>
              <div className="grid grid-cols-3 gap-3">
                <Input value={objectSyncForm.target_endpoint} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, target_endpoint: e.target.value })} placeholder="IP:端口 或 URL" />
                <Input value={objectSyncForm.target_region} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, target_region: e.target.value })} placeholder="region" />
                <Select value={String(objectSyncForm.target_use_ssl)} options={SSL_OPTIONS} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, target_use_ssl: e.target.value === 'true' })} />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <Input value={objectSyncForm.target_bucket} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, target_bucket: e.target.value })} placeholder="Bucket *" />
                <Input value={objectSyncForm.target_path} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, target_path: e.target.value })} placeholder="目标路径/前缀，可为空" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <Input value={objectSyncForm.target_access_key} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, target_access_key: e.target.value })} placeholder={editObjectSyncTarget ? 'Access Key 留空则不修改' : 'Access Key *'} />
                <Input type="password" value={objectSyncForm.target_secret_key} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, target_secret_key: e.target.value })} placeholder={editObjectSyncTarget ? 'Secret Key 留空则不修改' : 'Secret Key *'} />
              </div>
            </div>

            <div className="grid gap-2">
              <Label>描述</Label>
              <Input value={objectSyncForm.description} onChange={(e) => setObjectSyncForm({ ...objectSyncForm, description: e.target.value })} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowObjectSyncCreate(false); setEditObjectSyncTarget(null) }}>取消</Button>
            <Button onClick={editObjectSyncTarget ? handleUpdateObjectSync : handleCreateObjectSync} disabled={submitting}>
              {submitting ? '提交中...' : (editObjectSyncTarget ? '保存' : '创建')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirm */}
      <Dialog open={!!deleteTarget} onOpenChange={() => setDeleteTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>删除备份任务将同时清除所有历史记录，此操作不可撤销。</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>取消</Button>
            <Button variant="destructive" onClick={handleDelete} disabled={submitting}>
              {submitting ? '删除中...' : '确认删除'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Migration Delete Confirm */}
      <Dialog open={!!deleteMigrationTarget} onOpenChange={() => setDeleteMigrationTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>删除迁移任务会保留历史执行记录。</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteMigrationTarget(null)}>取消</Button>
            <Button variant="destructive" onClick={handleDeleteMigration} disabled={submitting}>
              {submitting ? '删除中...' : '确认删除'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Object Sync Delete Confirm */}
      <Dialog open={!!deleteObjectSyncTarget} onOpenChange={() => setDeleteObjectSyncTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>删除对象同步任务会保留历史执行记录。</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteObjectSyncTarget(null)}>取消</Button>
            <Button variant="destructive" onClick={handleDeleteObjectSync} disabled={submitting}>
              {submitting ? '删除中...' : '确认删除'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Migration Execute Confirm */}
      <Dialog open={!!executeMigrationTarget} onOpenChange={() => setExecuteMigrationTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>执行迁移任务</DialogTitle>
            <DialogDescription>
              {executeMigrationTarget?.mode === 'overwrite'
                ? '覆盖模式会在目标库存在时先删除再重建，请确认目标库允许被覆盖。'
                : '目标库不存在时会创建并导入，已存在的目标库会跳过。'}
            </DialogDescription>
          </DialogHeader>
          {executeMigrationTarget && (
            <div className="text-sm text-muted-foreground space-y-1">
              <p>任务：{executeMigrationTarget.name}</p>
              <p>源：{executeMigrationTarget.source_host}:{executeMigrationTarget.source_port}</p>
              <p>目标：{executeMigrationTarget.target_host}:{executeMigrationTarget.target_port}</p>
              <p>数据库：{formatBackupScope(executeMigrationTarget.db_names)}</p>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setExecuteMigrationTarget(null)}>取消</Button>
            <Button variant={executeMigrationTarget?.mode === 'overwrite' ? 'destructive' : 'default'} onClick={handleExecuteMigration} disabled={submitting}>
              {submitting ? '执行中...' : '确认执行'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Object Sync Execute Confirm */}
      <Dialog open={!!executeObjectSyncTarget} onOpenChange={() => setExecuteObjectSyncTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>执行对象同步任务</DialogTitle>
            <DialogDescription>
              {executeObjectSyncTarget?.mode === 'overwrite'
                ? '覆盖模式会覆盖目标路径下同名对象，请确认目标数据允许被写入。'
                : executeObjectSyncTarget?.mode === 'checksum_skip'
                  ? '系统会比较对象大小和 ETag，一致时跳过，不一致时覆盖写入。'
                  : '仅同步目标不存在的对象，目标已存在的同名对象会跳过。'}
            </DialogDescription>
          </DialogHeader>
          {executeObjectSyncTarget && (
            <div className="text-sm text-muted-foreground space-y-1">
              <p>任务：{executeObjectSyncTarget.name}</p>
              <p>源：{executeObjectSyncTarget.source_bucket}/{executeObjectSyncTarget.source_path || '-'}</p>
              <p>目标：{executeObjectSyncTarget.target_bucket}/{executeObjectSyncTarget.target_path || '-'}</p>
              <p>模式：{objectSyncModeLabel(executeObjectSyncTarget.mode)}</p>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setExecuteObjectSyncTarget(null)}>取消</Button>
            <Button variant={executeObjectSyncTarget?.mode === 'overwrite' ? 'destructive' : 'default'} onClick={handleExecuteObjectSync} disabled={submitting}>
              {submitting ? '执行中...' : '确认执行'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
