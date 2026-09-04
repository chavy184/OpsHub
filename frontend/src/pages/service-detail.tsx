import { useEffect, useState, useCallback, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useServiceStore } from '@/stores/service-store'
import { useReleaseStore } from '@/stores/release-store'
import { useHostStore } from '@/stores/host-store'
import { http } from '@/lib/http'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell,
} from '@/components/ui/table'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'
import { ArrowLeft, Plus, Search, Pencil, Trash2, Activity } from 'lucide-react'
import { Link } from 'react-router-dom'
import { CreateReleaseDialog } from '@/pages/releases'
import {
  ENV_CODE_OPTIONS, LOG_SOURCE_OPTIONS,
  buildLogSourceConfig, parseLogSourceConfig, LogSourceConfigEditor,
} from '@/components/env-form'
import type { ServiceEnv, ReleaseRecord, ReleaseStatus, CreateEnvPayload, UpdateEnvPayload, CreateReleasePayload } from '@/types/api'

// ─── 工具函数 ───────────────────────────────────────────

function healthBadgeVariant(status: string): 'default' | 'destructive' | 'secondary' {
  if (status === 'healthy') return 'default'
  if (status === 'unhealthy' || status === 'unreachable') return 'destructive'
  return 'secondary'
}

function healthLabel(status: string) {
  const map: Record<string, string> = { healthy: '健康', unhealthy: '异常', unreachable: '不可达', unknown: '未知' }
  return map[status] || status || '未知'
}

function releaseStatusBadge(status: ReleaseStatus) {
  const map: Record<ReleaseStatus, { label: string; variant: 'default' | 'destructive' | 'secondary' }> = {
    pending: { label: '待执行', variant: 'secondary' },
    running: { label: '执行中', variant: 'default' },
    success: { label: '成功', variant: 'default' },
    failed: { label: '失败', variant: 'destructive' },
    cancelled: { label: '已取消', variant: 'secondary' },
  }
  const item = map[status] || { label: status, variant: 'secondary' as const }
  return <Badge variant={item.variant}>{item.label}</Badge>
}

function formatDuration(start: string | null, end: string | null): string {
  if (!start || !end) return '-'
  const ms = new Date(end).getTime() - new Date(start).getTime()
  if (ms < 0) return '-'
  const sec = Math.floor(ms / 1000)
  if (sec < 60) return `${sec}s`
  return `${Math.floor(sec / 60)}m${sec % 60}s`
}

// ─── 主页面 ─────────────────────────────────────────────

export default function ServiceDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState('overview')

  const svc = useServiceStore((s) => s.currentService)
  const detailLoading = useServiceStore((s) => s.detailLoading)
  const fetchServiceDetail = useServiceStore((s) => s.fetchServiceDetail)

  useEffect(() => {
    if (id) fetchServiceDetail(id)
  }, [id, fetchServiceDetail])

  if (detailLoading || !svc) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-48 w-full" />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      {/* 顶部导航 */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate('/services')} aria-label="返回">
          <ArrowLeft className="size-4" />
        </Button>
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{svc.service_name}</h1>
          <p className="text-sm text-muted-foreground font-mono">{svc.service_key}</p>
        </div>
      </div>

      {/* Tab 导航 */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="overview">概览</TabsTrigger>
          <TabsTrigger value="envs">环境</TabsTrigger>
          <TabsTrigger value="hosts">主机</TabsTrigger>
          <TabsTrigger value="releases">发布历史</TabsTrigger>
          <TabsTrigger value="logs">日志</TabsTrigger>
        </TabsList>

        <TabsContent value="overview">
          <OverviewTab serviceId={svc.id} />
        </TabsContent>
        <TabsContent value="envs">
          <EnvsTab serviceId={svc.id} />
        </TabsContent>
        <TabsContent value="hosts">
          <HostsTab serviceId={svc.id} />
        </TabsContent>
        <TabsContent value="releases">
          <ReleasesTab serviceId={svc.id} />
        </TabsContent>
        <TabsContent value="logs">
          <LogsTab serviceId={svc.id} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

// ─── 概览 Tab ───────────────────────────────────────────

function OverviewTab({ serviceId }: { serviceId: string }) {
  const svc = useServiceStore((s) => s.currentService)!
  const envs = useServiceStore((s) => s.envs)
  const fetchEnvs = useServiceStore((s) => s.fetchEnvs)
  const [recentReleases, setRecentReleases] = useState<ReleaseRecord[]>([])
  const navigate = useNavigate()

  useEffect(() => {
    fetchEnvs(serviceId)
  }, [serviceId, fetchEnvs])

  useEffect(() => {
    http.get<{ list: ReleaseRecord[]; total: number }>('/api/v1/releases', {
      service_id: serviceId,
      page: 1,
      page_size: 3,
    }).then((res) => {
      setRecentReleases(res.data?.list ?? [])
    }).catch(() => {})
  }, [serviceId])

  return (
    <div className="flex flex-col gap-6 mt-4">
      {/* 基本信息 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">基本信息</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4 text-sm">
            <InfoRow label="服务标识" value={svc.service_key} />
            <InfoRow label="仓库地址" value={svc.repo_url || '-'} link={svc.repo_url || undefined} />
            <InfoRow label="负责人" value={svc.owner_user_id || '-'} />
          </div>
        </CardContent>
      </Card>

      {/* 环境健康状态 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Activity className="size-4" />
            环境健康状态
          </CardTitle>
        </CardHeader>
        <CardContent>
          {envs.length === 0 ? (
            <p className="text-sm text-muted-foreground">暂无环境配置</p>
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              {envs.map((env) => (
                <div key={env.id} className="rounded-lg border p-3 flex flex-col gap-1.5">
                  <span className="text-sm font-medium">{env.env_code}</span>
                  <Badge variant={healthBadgeVariant(env.health_status)} className="w-fit">
                    {healthLabel(env.health_status)}
                  </Badge>
                  {env.health_last_checked_at && (
                    <span className="text-xs text-muted-foreground">
                      上次检查: {env.health_last_checked_at}
                    </span>
                  )}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* 最近发布 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">最近发布</CardTitle>
        </CardHeader>
        <CardContent>
          {recentReleases.length === 0 ? (
            <p className="text-sm text-muted-foreground">暂无发布记录</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>类型</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>操作人</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead>耗时</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {recentReleases.map((r) => (
                  <TableRow
                    key={r.id}
                    className="cursor-pointer"
                    onClick={() => navigate(`/releases/${r.id}`)}
                  >
                    <TableCell>{r.release_type === 'deploy' ? '部署' : '回滚'}</TableCell>
                    <TableCell>{releaseStatusBadge(r.status)}</TableCell>
                    <TableCell>{r.operator_id || '-'}</TableCell>
                    <TableCell>{r.created_at}</TableCell>
                    <TableCell>{formatDuration(r.started_at, r.ended_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

// ─── 环境 Tab ───────────────────────────────────────────

function EnvsTab({ serviceId }: { serviceId: string }) {
  const envs = useServiceStore((s) => s.envs)
  const fetchEnvs = useServiceStore((s) => s.fetchEnvs)
  const createEnv = useServiceStore((s) => s.createEnv)
  const updateEnv = useServiceStore((s) => s.updateEnv)
  const deleteEnv = useServiceStore((s) => s.deleteEnv)
  const hosts = useHostStore((s) => s.hosts)
  const fetchHosts = useHostStore((s) => s.fetchHosts)

  const [loaded, setLoaded] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [editingEnv, setEditingEnv] = useState<ServiceEnv | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<ServiceEnv | null>(null)

  useEffect(() => {
    if (!loaded) {
      fetchEnvs(serviceId)
      fetchHosts()
      setLoaded(true)
    }
  }, [serviceId, fetchEnvs, fetchHosts, loaded])

  const handleDelete = async () => {
    if (!deleteConfirm) return
    try {
      await deleteEnv(serviceId, deleteConfirm.id)
      toast.success('环境已删除')
      setDeleteConfirm(null)
    } catch {
      toast.error('删除失败')
    }
  }

  return (
    <div className="flex flex-col gap-4 mt-4">
      <div className="flex items-center justify-between">
        <h2 className="text-base font-semibold">环境列表</h2>
        <Button size="sm" onClick={() => setCreateOpen(true)}>
          <Plus className="size-4 mr-1" />
          添加环境
        </Button>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>环境代码</TableHead>
            <TableHead>访问地址</TableHead>
            <TableHead>健康状态</TableHead>
            <TableHead>主机</TableHead>
            <TableHead>日志源</TableHead>
            <TableHead>Jenkins Job</TableHead>
            <TableHead>上次检查</TableHead>
            <TableHead className="w-24">操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {envs.length === 0 ? (
            <TableRow>
              <TableCell colSpan={8} className="text-center text-muted-foreground py-8">
                暂无环境
              </TableCell>
            </TableRow>
          ) : (
            envs.map((env) => (
              <TableRow key={env.id}>
                <TableCell className="font-medium">{env.env_code}</TableCell>
                <TableCell className="font-mono text-xs max-w-[200px] truncate">
                  {env.access_endpoint || '-'}
                </TableCell>
                <TableCell>
                  <Badge variant={healthBadgeVariant(env.health_status)}>
                    {healthLabel(env.health_status)}
                  </Badge>
                </TableCell>
                <TableCell className="font-mono text-xs">
                  {hosts.find((h) => h.id === env.host_id)?.name || env.host_id || '-'}
                </TableCell>
                <TableCell>{env.log_source_type || '-'}</TableCell>
                <TableCell className="font-mono text-xs max-w-[180px] truncate">
                  {(() => { try { const arr = JSON.parse(env.jenkins_jobs || '[]'); return arr[0]?.job || '-' } catch { return '-' } })()}
                </TableCell>
                <TableCell className="text-xs">{env.health_last_checked_at || '-'}</TableCell>
                <TableCell>
                  <div className="flex items-center gap-1">
                    <Button variant="ghost" size="icon" onClick={() => setEditingEnv(env)} aria-label="编辑">
                      <Pencil className="size-3.5" />
                    </Button>
                    <Button variant="ghost" size="icon" onClick={() => setDeleteConfirm(env)} aria-label="删除">
                      <Trash2 className="size-3.5" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>

      {/* 新增环境 */}
      <EnvCreateDialog
        open={createOpen}
        hosts={hosts}
        onOpenChange={setCreateOpen}
        onCreate={async (p) => { await createEnv(serviceId, p); toast.success('环境创建成功'); setCreateOpen(false) }}
      />

      {/* 编辑环境 */}
      {editingEnv && (
        <EnvEditDialog
          open={!!editingEnv}
          env={editingEnv}
          hosts={hosts}
          onOpenChange={(o) => { if (!o) setEditingEnv(null) }}
          onSave={async (p) => { await updateEnv(serviceId, editingEnv.id, p); toast.success('环境更新成功'); setEditingEnv(null) }}
        />
      )}

      {/* 删除确认 */}
      <Dialog open={!!deleteConfirm} onOpenChange={() => setDeleteConfirm(null)}>
        <DialogContent onClose={() => setDeleteConfirm(null)}>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>
              确定要删除环境 <strong>{deleteConfirm?.env_code}</strong> 吗？此操作不可撤销。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteConfirm(null)}>取消</Button>
            <Button variant="destructive" onClick={handleDelete}>删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

// ─── 环境新增对话框（与列表页一致） ─────────────────────

function EnvCreateDialog({ open, hosts, onOpenChange, onCreate }: {
  open: boolean; hosts: { id: string; name: string; host_address: string }[]; onOpenChange: (o: boolean) => void; onCreate: (p: CreateEnvPayload) => Promise<void>
}) {
  const [form, setForm] = useState<CreateEnvPayload>({ env_code: 'dev' })
  const [jenkinsJob, setJenkinsJob] = useState('')
  const [logFields, setLogFields] = useState<Record<string, string>>({})
  const [submitting, setSubmitting] = useState(false)
  const hostOptions = [{ value: '', label: '不关联主机' }, ...hosts.map((h) => ({ value: h.id, label: `${h.name} (${h.host_address})` }))]
  const handleSubmit = async () => {
    if (!form.env_code) { toast.error('环境编码为必填项'); return }
    if (!jenkinsJob) { toast.error('Jenkins Job 路径为必填'); return }
    setSubmitting(true)
    try {
      const configJson = form.log_source_type ? buildLogSourceConfig(form.log_source_type, logFields) : '{}'
      const jenkinsJobs = JSON.stringify([{ name: '构建', job: jenkinsJob }])
      await onCreate({ ...form, host_id: form.host_id || undefined, log_source_config: configJson, jenkins_jobs: jenkinsJobs })
      setForm({ env_code: 'dev' }); setLogFields({}); setJenkinsJob('')
    } catch (err) { toast.error(err instanceof Error ? err.message : '创建失败') }
    finally { setSubmitting(false) }
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg" onClose={() => onOpenChange(false)}>
        <DialogHeader><DialogTitle>新建环境</DialogTitle><DialogDescription>为服务添加新的运行环境</DialogDescription></DialogHeader>
        <div className="flex flex-col gap-3 py-2 max-h-[65vh] overflow-y-auto pr-1">
          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5"><Label>环境编码 *</Label><Select options={ENV_CODE_OPTIONS} value={form.env_code} onChange={(e) => setForm((f) => ({ ...f, env_code: e.target.value }))} /></div>
            <div className="flex flex-col gap-1.5"><Label>关联主机</Label><Select options={hostOptions} value={form.host_id ?? ''} onChange={(e) => setForm((f) => ({ ...f, host_id: e.target.value }))} /></div>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>Jenkins Job 路径 *</Label>
            <Input placeholder="如 folder/job-name" value={jenkinsJob} onChange={(e) => setJenkinsJob(e.target.value)} />
            <p className="text-xs text-muted-foreground">支持文件夹格式如 folder/job-name</p>
          </div>
          <div className="flex flex-col gap-1.5"><Label>访问地址</Label><Input placeholder="http://10.0.1.5:8080" value={form.access_endpoint ?? ''} onChange={(e) => setForm((f) => ({ ...f, access_endpoint: e.target.value }))} /></div>
          <div className="flex flex-col gap-1.5"><Label>健康检查 URL</Label><Input placeholder="http://10.0.1.5:8080/health" value={form.healthcheck_url ?? ''} onChange={(e) => setForm((f) => ({ ...f, healthcheck_url: e.target.value }))} /></div>
          <div className="flex flex-col gap-1.5">
            <Label>日志源类型</Label>
            <Select options={LOG_SOURCE_OPTIONS} value={form.log_source_type ?? ''} onChange={(e) => { setForm((f) => ({ ...f, log_source_type: e.target.value })); setLogFields({}) }} />
          </div>
          {form.log_source_type && <LogSourceConfigEditor type_={form.log_source_type} fields={logFields} onChange={setLogFields} />}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button onClick={handleSubmit} disabled={submitting}>{submitting ? '创建中…' : '创建'}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ─── 环境编辑对话框（与列表页一致） ─────────────────────

function EnvEditDialog({ open, env, hosts, onOpenChange, onSave }: {
  open: boolean; env: ServiceEnv; hosts: { id: string; name: string; host_address: string }[]; onOpenChange: (o: boolean) => void; onSave: (p: UpdateEnvPayload) => Promise<void>
}) {
  const existingJobs: { name: string; job: string }[] = (() => { try { return JSON.parse(env.jenkins_jobs || '[]') } catch { return [] } })()
  const [jenkinsJob, setJenkinsJob] = useState(existingJobs[0]?.job ?? '')
  const [form, setForm] = useState<UpdateEnvPayload>({
    host_id: env.host_id, access_endpoint: env.access_endpoint,
    healthcheck_url: env.healthcheck_url, log_source_type: env.log_source_type, log_source_config: env.log_source_config,
    health_check_interval: env.health_check_interval, health_check_timeout: env.health_check_timeout,
    health_check_success_codes: env.health_check_success_codes, health_check_enabled: env.health_check_enabled,
  })
  const [logFields, setLogFields] = useState<Record<string, string>>(() => parseLogSourceConfig(env.log_source_type ?? '', env.log_source_config ?? '{}'))
  const [submitting, setSubmitting] = useState(false)
  const hostOptions = [{ value: '', label: '不关联主机' }, ...hosts.map((h) => ({ value: h.id, label: `${h.name} (${h.host_address})` }))]
  const handleSave = async () => {
    if (!jenkinsJob) { toast.error('Jenkins Job 路径为必填'); return }
    setSubmitting(true)
    try {
      const configJson = form.log_source_type ? buildLogSourceConfig(form.log_source_type, logFields) : '{}'
      const jenkinsJobs = JSON.stringify([{ name: '构建', job: jenkinsJob }])
      await onSave({ ...form, host_id: form.host_id || undefined, log_source_config: configJson, jenkins_jobs: jenkinsJobs })
    } catch (err) { toast.error(err instanceof Error ? err.message : '保存失败') }
    finally { setSubmitting(false) }
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg" onClose={() => onOpenChange(false)}>
        <DialogHeader><DialogTitle>编辑环境 — {env.env_code}</DialogTitle><DialogDescription>修改环境配置信息</DialogDescription></DialogHeader>
        <div className="flex flex-col gap-3 py-2 max-h-[65vh] overflow-y-auto pr-1">
          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5"><Label>关联主机</Label><Select options={hostOptions} value={form.host_id ?? ''} onChange={(e) => setForm((f) => ({ ...f, host_id: e.target.value }))} /></div>
            <div className="flex flex-col gap-1.5"><Label>健康检查</Label>
              <Select options={[{ value: 'true', label: '启用' }, { value: 'false', label: '禁用' }]} value={form.health_check_enabled === false ? 'false' : 'true'} onChange={(e) => setForm((f) => ({ ...f, health_check_enabled: e.target.value === 'true' }))} />
            </div>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>Jenkins Job 路径 *</Label>
            <Input placeholder="如 folder/job-name" value={jenkinsJob} onChange={(e) => setJenkinsJob(e.target.value)} />
          </div>
          <div className="flex flex-col gap-1.5"><Label>访问地址</Label><Input value={form.access_endpoint ?? ''} onChange={(e) => setForm((f) => ({ ...f, access_endpoint: e.target.value }))} /></div>
          <div className="flex flex-col gap-1.5"><Label>健康检查 URL</Label><Input value={form.healthcheck_url ?? ''} onChange={(e) => setForm((f) => ({ ...f, healthcheck_url: e.target.value }))} /></div>
          <div className="flex flex-col gap-1.5">
            <Label>日志源类型</Label>
            <Select options={LOG_SOURCE_OPTIONS} value={form.log_source_type ?? ''} onChange={(e) => { const t = e.target.value; setForm((f) => ({ ...f, log_source_type: t })); setLogFields(parseLogSourceConfig(t, env.log_source_config ?? '{}')) }} />
          </div>
          {form.log_source_type && <LogSourceConfigEditor type_={form.log_source_type} fields={logFields} onChange={setLogFields} />}
          {form.health_check_enabled !== false && (
            <div className="grid grid-cols-2 gap-3">
              <div className="flex flex-col gap-1.5"><Label>检查间隔 (秒)</Label><Input type="number" value={form.health_check_interval ?? 60} onChange={(e) => setForm((f) => ({ ...f, health_check_interval: Number(e.target.value) }))} /></div>
              <div className="flex flex-col gap-1.5"><Label>超时 (秒)</Label><Input type="number" value={form.health_check_timeout ?? 10} onChange={(e) => setForm((f) => ({ ...f, health_check_timeout: Number(e.target.value) }))} /></div>
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button onClick={handleSave} disabled={submitting}>{submitting ? '保存中…' : '保存'}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ─── 发布历史 Tab ───────────────────────────────────────

function ReleasesTab({ serviceId }: { serviceId: string }) {
  const releases = useReleaseStore((s) => s.releases)
  const total = useReleaseStore((s) => s.total)
  const loading = useReleaseStore((s) => s.loading)
  const setQuery = useReleaseStore((s) => s.setQuery)
  const fetchReleases = useReleaseStore((s) => s.fetchReleases)
  const createRelease = useReleaseStore((s) => s.createRelease)
  const navigate = useNavigate()
  const [statusFilter, setStatusFilter] = useState('')
  const [initialized, setInitialized] = useState(false)
  const [showCreate, setShowCreate] = useState(false)

  useEffect(() => {
    if (!initialized) {
      setQuery({ service_id: serviceId, page: 1, page_size: 20, status: undefined })
      setInitialized(true)
    }
  }, [serviceId, setQuery, initialized])

  const handleStatusChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const val = e.target.value
    setStatusFilter(val)
    setQuery({
      service_id: serviceId,
      status: (val || undefined) as ReleaseStatus | undefined,
      page: 1,
    })
  }

  const releaseStatusInfo = (status: ReleaseStatus): { label: string; variant: 'default' | 'destructive' | 'secondary' | 'outline' } => {
    switch (status) {
      case 'pending': return { label: '待执行', variant: 'secondary' }
      case 'running': return { label: '执行中', variant: 'default' }
      case 'success': return { label: '成功', variant: 'default' }
      case 'failed':  return { label: '失败', variant: 'destructive' }
      case 'cancelled': return { label: '已取消', variant: 'outline' }
      default: return { label: status, variant: 'secondary' }
    }
  }

  return (
    <div className="flex flex-col gap-4 mt-4">
      <div className="flex items-center justify-between">
        <h2 className="text-base font-semibold">发布历史</h2>
        <div className="flex items-center gap-2">
          <Select
            options={[
              { value: '', label: '全部状态' },
              { value: 'pending', label: '待执行' },
              { value: 'running', label: '执行中' },
              { value: 'success', label: '成功' },
              { value: 'failed', label: '失败' },
              { value: 'cancelled', label: '已取消' },
            ]}
            value={statusFilter}
            onChange={handleStatusChange}
            className="w-32"
          />
          <Button size="sm" onClick={() => setShowCreate(true)}>
            <Plus className="size-4 mr-1" />
            新建发布
          </Button>
        </div>
      </div>

      {loading ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>环境</TableHead>
                <TableHead>类型</TableHead>
                <TableHead>策略</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>操作人</TableHead>
                <TableHead>开始时间</TableHead>
                <TableHead>耗时</TableHead>
                <TableHead className="max-w-48">错误信息</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {releases.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={9} className="text-center text-muted-foreground py-8">
                    暂无发布记录
                  </TableCell>
                </TableRow>
              ) : (
                releases.map((r) => {
                  const info = releaseStatusInfo(r.status)
                  return (
                    <TableRow
                      key={r.id}
                      className="cursor-pointer hover:bg-muted/40"
                      onClick={() => navigate(`/releases/${r.id}?from_service=${serviceId}`)}
                    >
                      <TableCell className="font-mono text-xs text-primary">{r.id.slice(0, 8)}</TableCell>
                      <TableCell className="text-xs">{r.env_code || '-'}</TableCell>
                      <TableCell className="text-xs">{r.release_type === 'deploy' ? '部署' : '回滚'}</TableCell>
                      <TableCell className="text-xs">{r.strategy}</TableCell>
                      <TableCell><Badge variant={info.variant}>{info.label}</Badge></TableCell>
                      <TableCell className="text-xs">{r.operator_id || '-'}</TableCell>
                      <TableCell className="text-xs">{r.started_at ?? '-'}</TableCell>
                      <TableCell className="text-xs">{formatDuration(r.started_at, r.ended_at)}</TableCell>
                      <TableCell className="max-w-48 truncate text-xs text-destructive">{r.error_message || '-'}</TableCell>
                    </TableRow>
                  )
                })
              )}
            </TableBody>
          </Table>
        </div>
      )}

      {total > 20 && (
        <p className="text-xs text-muted-foreground text-center">
          共 {total} 条记录，前往发布管理页面查看更多
        </p>
      )}

      <CreateReleaseDialog
        open={showCreate}
        onOpenChange={setShowCreate}
        defaultServiceId={serviceId}
        lockService
        onCreate={async (p: CreateReleasePayload) => {
          const rec = await createRelease(p)
          // createRelease 内部已调用 fetchReleases，这里不需重复
          void fetchReleases
          return rec
        }}
      />
    </div>
  )
}

// ─── 日志 Tab ───────────────────────────────────────────

interface LogLine { ts: number; line: string }

const LEVEL_OPTIONS = [
  { value: '', label: '全部级别' },
  { value: 'ERROR', label: 'ERROR' },
  { value: 'WARN', label: 'WARN' },
  { value: 'INFO', label: 'INFO' },
  { value: 'DEBUG', label: 'DEBUG' },
]

function LogsTab({ serviceId }: { serviceId: string }) {
  const envs = useServiceStore((s) => s.envs)
  const fetchEnvs = useServiceStore((s) => s.fetchEnvs)

  const [envId, setEnvId] = useState('')
  const [keyword, setKeyword] = useState('')
  const [level, setLevel] = useState('')
  const [startTime, setStartTime] = useState('')
  const [endTime, setEndTime] = useState('')
  const [loading, setLoading] = useState(false)
  const [lines, setLines] = useState<LogLine[]>([])
  const [searched, setSearched] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedContainer, setSelectedContainer] = useState('')
  const [selectedLabelSet, setSelectedLabelSet] = useState('')
  const [selectedFile, setSelectedFile] = useState('')

  useEffect(() => {
    fetchEnvs(serviceId)
  }, [serviceId, fetchEnvs])

  // Docker 多容器解析
  const selectedEnv = envs.find((e) => e.id === envId)
  const dockerContainers = useMemo(() => {
    if (selectedEnv?.log_source_type !== 'docker' || !selectedEnv?.log_source_config) return []
    try {
      const obj = JSON.parse(selectedEnv.log_source_config)
      if (Array.isArray(obj.containers) && obj.containers.length > 0) {
        return obj.containers as Array<{ name: string; container: string }>
      }
      if (typeof obj.container === 'string' && obj.container) {
        return [{ name: 'default', container: obj.container }]
      }
    } catch {}
    return []
  }, [selectedEnv])

  // Loki 多标签值解析
  const lokiLabelValues = useMemo(() => {
    if (selectedEnv?.log_source_type !== 'loki' || !selectedEnv?.log_source_config) return []
    try {
      const obj = JSON.parse(selectedEnv.log_source_config)
      if (Array.isArray(obj.label_values) && obj.label_values.length > 0) {
        return obj.label_values as string[]
      }
      // 旧格式: labels 对象
      if (obj.labels && typeof obj.labels === 'object') {
        return Object.values(obj.labels as Record<string, string>) as string[]
      }
    } catch {}
    return []
  }, [selectedEnv])

  // File 多文件解析
  const fileSources = useMemo(() => {
    if (selectedEnv?.log_source_type !== 'file' || !selectedEnv?.log_source_config) return []
    try {
      const obj = JSON.parse(selectedEnv.log_source_config)
      if (Array.isArray(obj.files) && obj.files.length > 0) {
        return obj.files as Array<{ name: string; path: string }>
      }
      if (typeof obj.path === 'string' && obj.path) {
        return [{ name: 'default', path: obj.path }]
      }
    } catch {}
    return []
  }, [selectedEnv])

  // 环境变化时重置选中
  useEffect(() => {
    if (dockerContainers.length > 0) setSelectedContainer(dockerContainers[0]!.container)
    else setSelectedContainer('')
  }, [dockerContainers])
  useEffect(() => {
    if (lokiLabelValues.length > 0) setSelectedLabelSet(lokiLabelValues[0]!)
    else setSelectedLabelSet('')
  }, [lokiLabelValues])
  useEffect(() => {
    if (fileSources.length > 0) setSelectedFile(fileSources[0]!.path)
    else setSelectedFile('')
  }, [fileSources])

  const envOptions = [
    { value: '', label: '请选择环境' },
    ...envs.map((e) => ({ value: e.id, label: e.env_code })),
  ]

  const handleSearch = useCallback(async () => {
    if (!envId) {
      toast.error('请先选择环境')
      return
    }
    setLoading(true)
    setError(null)
    setSearched(true)
    setLines([])
    try {
      const body = {
        serviceId,
        envId,
        container: selectedContainer || undefined,
        file: selectedFile || undefined,
        labelSet: selectedLabelSet || undefined,
        keyword,
        level,
        startTime: startTime || undefined,
        endTime: endTime || undefined,
        limit: 200,
      }
      const response = await http.post<any>('/api/v1/logs/search', body)
      const lokiData = response.data?.data
      const parsed: LogLine[] = []
      if (lokiData?.result && Array.isArray(lokiData.result)) {
        for (const stream of lokiData.result) {
          if (!stream?.values || !Array.isArray(stream.values)) continue
          for (const [tsNano, line] of stream.values) {
            parsed.push({ ts: Number(tsNano) / 1e6, line })
          }
        }
      }
      parsed.sort((a, b) => b.ts - a.ts)
      setLines(parsed)
    } catch (err: any) {
      setError(err?.message || '日志查询失败')
    } finally {
      setLoading(false)
    }
  }, [serviceId, envId, keyword, level, startTime, endTime, selectedContainer, selectedLabelSet, selectedFile])

  return (
    <div className="flex flex-col gap-4 mt-4">
      <div className="rounded-lg border p-4 flex flex-col gap-4">
        <div className="grid grid-cols-2 gap-4">
          <div className="flex flex-col gap-1.5">
            <Label>环境</Label>
            <Select
              options={envOptions}
              value={envId}
              onChange={(e) => setEnvId(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>日志级别</Label>
            <Select
              options={LEVEL_OPTIONS}
              value={level}
              onChange={(e) => setLevel(e.target.value)}
            />
          </div>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div className="flex flex-col gap-1.5">
            <Label>开始时间</Label>
            <Input
              type="datetime-local"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>结束时间</Label>
            <Input
              type="datetime-local"
              value={endTime}
              onChange={(e) => setEndTime(e.target.value)}
            />
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Input
            className="flex-1"
            placeholder="关键词搜索（支持正则）"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          />
          <Button onClick={handleSearch} disabled={loading || !envId}>
            <Search className="size-4 mr-1" />
            {loading ? '搜索中...' : '搜索'}
          </Button>
        </div>

        {/* Docker 多容器 Tab 切换 */}
        {dockerContainers.length > 1 && (
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground">容器:</span>
            <Tabs value={selectedContainer} onValueChange={setSelectedContainer}>
              <TabsList>
                {dockerContainers.map((ct) => (
                  <TabsTrigger key={ct.container} value={ct.container}>
                    {ct.name || ct.container}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          </div>
        )}

        {/* Loki 多标签值 Tab 切换 */}
        {lokiLabelValues.length > 1 && (
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground">标签:</span>
            <Tabs value={selectedLabelSet} onValueChange={setSelectedLabelSet}>
              <TabsList>
                {lokiLabelValues.map((val) => (
                  <TabsTrigger key={val} value={val}>
                    {val}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          </div>
        )}

        {/* File 多文件 Tab 切换 */}
        {fileSources.length > 1 && (
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground">文件:</span>
            <Tabs value={selectedFile} onValueChange={setSelectedFile}>
              <TabsList>
                {fileSources.map((f) => (
                  <TabsTrigger key={f.path} value={f.path}>
                    {f.name || f.path}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          </div>
        )}
      </div>

      {/* 结果区域 */}
      {!searched ? (
        <div className="flex flex-col items-center gap-3 py-16 text-muted-foreground">
          <Search className="size-10 opacity-20" />
          <p className="text-sm">选择环境后点击搜索查看日志</p>
        </div>
      ) : error ? (
        <div className="rounded-lg border border-destructive/50 p-4 text-sm text-destructive">
          {error}
        </div>
      ) : lines.length === 0 ? (
        <div className="flex flex-col items-center gap-2 py-16 text-muted-foreground">
          <p className="text-sm">未找到匹配的日志</p>
        </div>
      ) : (
        <div className="rounded-lg border">
          <div className="p-2 border-b text-xs text-muted-foreground">
            共 {lines.length} 条日志
          </div>
          <div className="max-h-[500px] overflow-auto">
            {lines.map((l, i) => (
              <div
                key={i}
                className="px-3 py-1 text-xs font-mono border-b last:border-0 hover:bg-muted/50"
              >
                <span className="text-muted-foreground mr-2">
                  {new Date(l.ts).toLocaleString()}
                </span>
                <span className="whitespace-pre-wrap break-all">{l.line}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

// ─── 共用组件 ───────────────────────────────────────────

function InfoRow({ label, value, link }: { label: string; value: string; link?: string }) {
  const isUrl = link && /^https?:\/\//i.test(link)
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-muted-foreground">{label}</span>
      {isUrl ? (
        <a
          href={link}
          target="_blank"
          rel="noopener noreferrer"
          className="font-medium text-primary hover:underline truncate"
          title={link}
        >
          {value}
        </a>
      ) : (
        <span className="font-medium truncate" title={value}>{value}</span>
      )}
    </div>
  )
}

// ─── 告警 Tab ───────────────────────────────────────────
// 此模块已下线，AlertsTab/alertSeverityBadge 暂不挂载到 UI

// ─── 主机 Tab ───────────────────────────────────────────

function HostsTab({ serviceId }: { serviceId: string }) {
  const envs = useServiceStore((s) => s.envs)
  const fetchEnvs = useServiceStore((s) => s.fetchEnvs)
  const hosts = useHostStore((s) => s.hosts)
  const fetchHosts = useHostStore((s) => s.fetchHosts)

  useEffect(() => {
    fetchEnvs(serviceId)
    if (hosts.length === 0) fetchHosts()
  }, [serviceId, fetchEnvs, fetchHosts, hosts.length])

  const hostMap = new Map(hosts.map((h) => [h.id, h]))
  const boundEnvs = envs.filter((e) => e.host_id)
  const hostIds = Array.from(new Set(boundEnvs.map((e) => e.host_id).filter(Boolean) as string[]))

  return (
    <Card className="mt-4">
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-sm">该服务部署的主机（{hostIds.length}）</CardTitle>
        <Link to="/hosts">
          <Button size="sm" variant="outline">主机管理</Button>
        </Link>
      </CardHeader>
      <CardContent>
        {boundEnvs.length === 0 ? (
          <p className="text-sm text-muted-foreground py-6 text-center">未绑定主机</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>环境</TableHead>
                <TableHead>主机名</TableHead>
                <TableHead>IP</TableHead>
                <TableHead>访问入口</TableHead>
                <TableHead>健康</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {boundEnvs.map((e) => {
                const h = hostMap.get(e.host_id!)
                return (
                  <TableRow key={e.id}>
                    <TableCell><Badge variant="outline">{e.env_code}</Badge></TableCell>
                    <TableCell className="text-sm">{h?.name || '-'}</TableCell>
                    <TableCell className="font-mono text-xs">{h?.host_address || '-'}</TableCell>
                    <TableCell className="font-mono text-xs">{e.access_endpoint || '-'}</TableCell>
                    <TableCell>
                      <Badge variant={healthBadgeVariant(e.health_status || 'unknown')}>
                        {healthLabel(e.health_status || 'unknown')}
                      </Badge>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}