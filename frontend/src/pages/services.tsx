import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useServiceStore } from '@/stores/service-store'
import { useHostStore } from '@/stores/host-store'
import { listEnvs } from '@/api/services'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Select } from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Card } from '@/components/ui/card'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Plus, Search, ExternalLink, ScrollText, Pencil, Trash2,
  Server, Package, Layers,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  ENV_CODE_OPTIONS, LOG_SOURCE_OPTIONS,
  buildLogSourceConfig, parseLogSourceConfig, LogSourceConfigEditor,
} from '@/components/env-form'
import type {
  Service, ServiceEnv, CreateServicePayload, UpdateServicePayload,
  CreateEnvPayload, UpdateEnvPayload, Host,
} from '@/types/api'

// 环境颜色：使用饱和的实心 Badge，提升识别度
const ENV_TONE: Record<string, string> = {
  prod:    'border-transparent bg-rose-500 text-white shadow-sm shadow-rose-500/20 dark:bg-rose-600',
  staging: 'border-transparent bg-amber-500 text-white shadow-sm shadow-amber-500/20 dark:bg-amber-600',
  uat:     'border-transparent bg-blue-500 text-white shadow-sm shadow-blue-500/20 dark:bg-blue-600',
  dev:     'border-transparent bg-emerald-500 text-white shadow-sm shadow-emerald-500/20 dark:bg-emerald-600',
  test:    'border-transparent bg-violet-500 text-white shadow-sm shadow-violet-500/20 dark:bg-violet-600',
}

function envTone(code: string): string {
  return ENV_TONE[code.toLowerCase()] || 'border-transparent bg-slate-500 text-white shadow-sm'
}

// 环境健康状态点颜色 + 文案
function envHealthDot(env: ServiceEnv): { dotClass: string; label: string } {
  if (!env.health_check_enabled) {
    return { dotClass: 'bg-slate-300 dark:bg-slate-600', label: '未开启检查' }
  }
  const s = (env.health_status || '').toLowerCase()
  if (s === 'healthy')   return { dotClass: 'bg-emerald-500 shadow-[0_0_6px] shadow-emerald-500/60', label: '健康' }
  if (s === 'unhealthy' || s === 'unreachable')
                         return { dotClass: 'bg-rose-500 shadow-[0_0_6px] shadow-rose-500/60', label: s === 'unreachable' ? '不可达' : '异常' }
  return { dotClass: 'bg-slate-300 dark:bg-slate-600', label: '未知' }
}

export default function ServicesPage() {
  const navigate = useNavigate()
  const services = useServiceStore((s) => s.services)
  const total = useServiceStore((s) => s.total)
  const loading = useServiceStore((s) => s.loading)
  const query = useServiceStore((s) => s.query)
  const fetchServices = useServiceStore((s) => s.fetchServices)
  const setQuery = useServiceStore((s) => s.setQuery)
  const createService = useServiceStore((s) => s.createService)
  const updateService = useServiceStore((s) => s.updateService)
  const deleteService = useServiceStore((s) => s.deleteService)
  const createEnv = useServiceStore((s) => s.createEnv)
  const updateEnv = useServiceStore((s) => s.updateEnv)
  const deleteEnv = useServiceStore((s) => s.deleteEnv)
  const hosts = useHostStore((s) => s.hosts)
  const fetchHosts = useHostStore((s) => s.fetchHosts)

  const [searchInput, setSearchInput] = useState('')
  const [envsMap, setEnvsMap] = useState<Record<string, ServiceEnv[]>>({})
  const [loadingEnvIds, setLoadingEnvIds] = useState<Set<string>>(new Set())
  const [showCreate, setShowCreate] = useState(false)
  const [editingService, setEditingService] = useState<Service | null>(null)
  const [deletingServiceId, setDeletingServiceId] = useState<string | null>(null)
  const [addEnvServiceId, setAddEnvServiceId] = useState<string | null>(null)
  const [editingEnv, setEditingEnv] = useState<{ env: ServiceEnv; serviceId: string } | null>(null)
  const [deletingEnv, setDeletingEnv] = useState<{ envId: string; serviceId: string } | null>(null)

  const loadEnvs = useCallback(async (serviceId: string) => {
    setLoadingEnvIds((s) => { const ns = new Set(s); ns.add(serviceId); return ns })
    try {
      const envs = await listEnvs(serviceId)
      setEnvsMap((m) => ({ ...m, [serviceId]: Array.isArray(envs) ? envs : [] }))
    } catch {
      setEnvsMap((m) => ({ ...m, [serviceId]: [] }))
    } finally {
      setLoadingEnvIds((s) => { const ns = new Set(s); ns.delete(serviceId); return ns })
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => { fetchServices(); fetchHosts() }, [fetchServices, fetchHosts])

  // 卡片视图：服务列表加载完成后批量预载各服务的环境
  useEffect(() => {
    services.forEach((s) => { if (envsMap[s.id] === undefined && !loadingEnvIds.has(s.id)) loadEnvs(s.id) })
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [services])

  const refreshEnvs = async (serviceId: string) => {
    setEnvsMap((m) => { const nm = { ...m }; delete nm[serviceId]; return nm })
    await loadEnvs(serviceId)
  }

  const handleDeleteService = async () => {
    if (!deletingServiceId) return
    try { await deleteService(deletingServiceId); toast.success('服务已删除') }
    catch (err) { toast.error(err instanceof Error ? err.message : '删除失败') }
    finally { setDeletingServiceId(null) }
  }

  const handleDeleteEnv = async () => {
    if (!deletingEnv) return
    try {
      await deleteEnv(deletingEnv.serviceId, deletingEnv.envId)
      await refreshEnvs(deletingEnv.serviceId)
      toast.success('环境已删除')
    } catch (err) { toast.error(err instanceof Error ? err.message : '删除失败') }
    finally { setDeletingEnv(null) }
  }

  return (
    <div className="flex flex-col gap-6">
      {/* 顶部标题 + 统计 */}
      <div className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <Server className="size-6 text-primary" />
            服务管理
          </h1>
          <p className="text-muted-foreground text-sm mt-1">管理所有服务及其多环境部署信息</p>
        </div>
        <Button onClick={() => setShowCreate(true)} className="gap-1">
          <Plus className="size-4" />新建服务
        </Button>
      </div>

      {/* 搜索栏 */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground pointer-events-none" />
          <Input
            placeholder="搜索服务名称 / 服务标识…"
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && setQuery({ keyword: searchInput, page: 1 })}
            className="pl-9"
          />
        </div>
        <Button variant="outline" size="sm" onClick={() => setQuery({ keyword: searchInput, page: 1 })}>
          搜索
        </Button>
        {query.keyword && (
          <Button variant="ghost" size="sm" onClick={() => { setSearchInput(''); setQuery({ keyword: '', page: 1 }) }}>
            清除
          </Button>
        )}
      </div>

      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-44 w-full rounded-lg" />)}
        </div>
      ) : services.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed py-20 text-muted-foreground">
          <Package className="size-10 opacity-30" />
          <p className="text-sm">{query.keyword ? '未匹配到服务' : '暂无服务数据'}</p>
          {!query.keyword && (
            <Button variant="outline" onClick={() => setShowCreate(true)} className="mt-2 gap-1">
              <Plus className="size-3.5" />创建第一个服务
            </Button>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {services.map((svc) => {
            const envs = envsMap[svc.id] ?? []
            const envLoading = loadingEnvIds.has(svc.id)
            return (
              <ServiceCard
                key={svc.id}
                svc={svc}
                envs={envs}
                envLoading={envLoading}
                onOpenDetail={() => navigate(`/services/${svc.id}`)}
                onEdit={() => setEditingService(svc)}
                onDelete={() => setDeletingServiceId(svc.id)}
                onAddEnv={() => setAddEnvServiceId(svc.id)}
                onEditEnv={(env) => setEditingEnv({ env, serviceId: svc.id })}
                onDeleteEnv={(env) => setDeletingEnv({ envId: env.id, serviceId: svc.id })}
                onViewLogs={(env) => navigate(`/logs?serviceId=${svc.id}&envId=${env.id}`)}
              />
            )
          })}
        </div>
      )}

      {total > query.page_size && (
        <div className="flex items-center justify-end gap-2">
          <Button variant="outline" size="sm" disabled={query.page <= 1} onClick={() => setQuery({ page: query.page - 1 })}>上一页</Button>
          <span className="text-sm text-muted-foreground">{query.page} / {Math.ceil(total / query.page_size)}</span>
          <Button variant="outline" size="sm" disabled={query.page >= Math.ceil(total / query.page_size)} onClick={() => setQuery({ page: query.page + 1 })}>下一页</Button>
        </div>
      )}

      <CreateServiceDialog open={showCreate} onOpenChange={setShowCreate} onCreate={createService} />
      {editingService && (
        <EditServiceDialog open={!!editingService} service={editingService}
          onOpenChange={(o) => { if (!o) setEditingService(null) }}
          onSave={async (p) => { await updateService(editingService.id, p); setEditingService(null) }} />
      )}
      <Dialog open={!!deletingServiceId} onOpenChange={(o) => { if (!o) setDeletingServiceId(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>确认删除服务</DialogTitle><DialogDescription>该服务及所有环境配置将被永久删除，无法恢复。</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeletingServiceId(null)}>取消</Button>
            <Button variant="destructive" onClick={handleDeleteService}>确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      {addEnvServiceId && (
        <CreateEnvDialog open={!!addEnvServiceId} hosts={hosts} onOpenChange={(o) => { if (!o) setAddEnvServiceId(null) }}
          onCreate={async (p) => { await createEnv(addEnvServiceId, p); await refreshEnvs(addEnvServiceId); setAddEnvServiceId(null) }} />
      )}
      {editingEnv && (
        <EditEnvDialog open={!!editingEnv} env={editingEnv.env} hosts={hosts}
          onOpenChange={(o) => { if (!o) setEditingEnv(null) }}
          onSave={async (p) => { await updateEnv(editingEnv.serviceId, editingEnv.env.id, p); await refreshEnvs(editingEnv.serviceId); setEditingEnv(null) }} />
      )}
      <Dialog open={!!deletingEnv} onOpenChange={(o) => { if (!o) setDeletingEnv(null) }}>
        <DialogContent>
          <DialogHeader><DialogTitle>确认删除环境</DialogTitle><DialogDescription>删除后该环境的所有配置将不可恢复。</DialogDescription></DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeletingEnv(null)}>取消</Button>
            <Button variant="destructive" onClick={handleDeleteEnv}>确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

// 服务卡片
function ServiceCard({
  svc, envs, envLoading,
  onOpenDetail, onEdit, onDelete, onAddEnv, onEditEnv, onDeleteEnv, onViewLogs,
}: {
  svc: Service
  envs: ServiceEnv[]
  envLoading: boolean
  onOpenDetail: () => void
  onEdit: () => void
  onDelete: () => void
  onAddEnv: () => void
  onEditEnv: (env: ServiceEnv) => void
  onDeleteEnv: (env: ServiceEnv) => void
  onViewLogs: (env: ServiceEnv) => void
}) {
  return (
    <Card className="group relative flex flex-col overflow-hidden border-border/60 transition-all hover:border-primary/40 hover:shadow-md">
      {/* 头部：名称 + 关键属性 */}
      <div className="flex items-start gap-3 p-4 pb-3">
        <div className="flex shrink-0 size-10 items-center justify-center rounded-md border bg-muted/50 font-semibold text-sm">
          {svc.service_name.slice(0, 1).toUpperCase()}
        </div>
        <div className="min-w-0 flex-1">
          <button
            type="button"
            onClick={onOpenDetail}
            className="text-left font-semibold truncate hover:text-primary transition-colors block w-full"
            title={svc.service_name}
          >
            {svc.service_name}
          </button>
          <div className="font-mono text-xs text-muted-foreground truncate">{svc.service_key}</div>
        </div>
        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={onEdit} title="编辑">
            <Pencil className="size-3.5" />
          </Button>
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0 text-destructive hover:text-destructive hover:bg-destructive/10" onClick={onDelete} title="删除">
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      </div>

      {/* 元数据栏 */}
      <div className="flex items-center gap-2 px-4 pb-3 text-xs text-muted-foreground">
        {envs.length > 0 ? (
          <span className="inline-flex items-center gap-1">
            <Layers className="size-3" />{envs.length} 个环境
          </span>
        ) : (
          <span className="italic">暂无环境</span>
        )}
      </div>

      {/* 环境列表 */}
      <div className="border-t border-border/60 bg-muted/20 p-3 flex-1">
        {envLoading ? (
          <div className="flex flex-wrap gap-1.5">
            <Skeleton className="h-7 w-20" />
            <Skeleton className="h-7 w-24" />
          </div>
        ) : envs.length === 0 ? (
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span className="italic">暂无环境</span>
            <Button variant="ghost" size="sm" className="h-7 text-xs gap-1" onClick={onAddEnv}>
              <Plus className="size-3" />添加
            </Button>
          </div>
        ) : (
          <div className="flex flex-col gap-1.5">
            {envs.map((env) => {
              const health = envHealthDot(env)
              return (
                <div
                  key={env.id}
                  className="group/env flex items-center gap-2 rounded-md border border-transparent bg-background/60 px-2 py-1.5 hover:border-border hover:bg-background transition-colors"
                >
                  <span
                    className={`inline-block size-2 shrink-0 rounded-full ${health.dotClass}`}
                    title={`健康状态：${health.label}`}
                  />
                  <Badge variant="outline" className={`font-mono text-xs font-semibold uppercase tracking-wide px-2 py-0 h-5 ${envTone(env.env_code)}`}>
                    {env.env_code}
                  </Badge>
                  {env.access_endpoint && (
                    <span className="text-[11px] text-muted-foreground/80 font-mono truncate flex-1" title={env.access_endpoint}>
                      {env.access_endpoint}
                    </span>
                  )}
                  <div className="ml-auto flex items-center gap-0.5 opacity-0 group-hover/env:opacity-100 transition-opacity">
                    {env.log_source_type && (
                      <Button variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={() => onViewLogs(env)} title="查看日志">
                        <ScrollText className="size-3" />
                      </Button>
                    )}
                    <Button variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={() => onEditEnv(env)} title="编辑环境">
                      <Pencil className="size-3" />
                    </Button>
                    <Button variant="ghost" size="sm" className="h-6 w-6 p-0 text-destructive hover:text-destructive hover:bg-destructive/10" onClick={() => onDeleteEnv(env)} title="删除环境">
                      <Trash2 className="size-3" />
                    </Button>
                  </div>
                </div>
              )
            })}
            <Button variant="ghost" size="sm" className="h-7 text-xs gap-1 justify-start text-muted-foreground hover:text-foreground" onClick={onAddEnv}>
              <Plus className="size-3" />新增环境
            </Button>
          </div>
        )}
      </div>

      {/* 底部：详情入口 */}
      <button
        type="button"
        onClick={onOpenDetail}
        className="flex items-center justify-between gap-1 border-t border-border/60 px-4 py-2 text-xs text-muted-foreground hover:bg-muted/40 hover:text-foreground transition-colors"
      >
        <span>查看详情 / 发布 / 监控</span>
        <ExternalLink className="size-3.5" />
      </button>
    </Card>
  )
}

// 顶部统计卡片
function CreateServiceDialog({ open, onOpenChange, onCreate }: {
  open: boolean; onOpenChange: (o: boolean) => void; onCreate: (p: CreateServicePayload) => Promise<unknown>
}) {
  const [form, setForm] = useState<{ service_key: string; service_name: string; repo_url?: string }>({ service_key: '', service_name: '' })
  const [submitting, setSubmitting] = useState(false)
  const handleSubmit = async () => {
    if (!form.service_key || !form.service_name) { toast.error('服务标识和名称为必填项'); return }
    setSubmitting(true)
    try {
      const payload: CreateServicePayload = {
        service_key: form.service_key,
        service_name: form.service_name,
        repo_url: form.repo_url,
      }
      await onCreate(payload); toast.success('服务创建成功'); onOpenChange(false); setForm({ service_key: '', service_name: '' })
    }
    catch (err) { toast.error(err instanceof Error ? err.message : '创建失败') }
    finally { setSubmitting(false) }
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader><DialogTitle>新建服务</DialogTitle></DialogHeader>
        <div className="flex flex-col gap-4 py-4">
          <div className="flex flex-col gap-1.5">
            <Label>服务标识 *</Label>
            <Input placeholder="如 user-service" value={form.service_key} onChange={(e) => setForm((f) => ({ ...f, service_key: e.target.value }))} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>服务名称 *</Label>
            <Input placeholder="如 用户服务" value={form.service_name} onChange={(e) => setForm((f) => ({ ...f, service_name: e.target.value }))} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>仓库地址</Label>
            <Input placeholder="https://github.com/..." value={form.repo_url ?? ''} onChange={(e) => setForm((f) => ({ ...f, repo_url: e.target.value }))} />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button onClick={handleSubmit} disabled={submitting}>{submitting ? '创建中...' : '创建'}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function EditServiceDialog({ open, service, onOpenChange, onSave }: {
  open: boolean; service: Service; onOpenChange: (o: boolean) => void; onSave: (p: UpdateServicePayload) => Promise<void>
}) {
  const [form, setForm] = useState<UpdateServicePayload>({ service_name: service.service_name, repo_url: service.repo_url })
  const [submitting, setSubmitting] = useState(false)
  const handleSave = async () => {
    if (!form.service_name) { toast.error('服务名称不能为空'); return }
    setSubmitting(true)
    try { await onSave(form); toast.success('服务已更新'); onOpenChange(false) }
    catch (err) { toast.error(err instanceof Error ? err.message : '更新失败') }
    finally { setSubmitting(false) }
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader><DialogTitle>编辑服务 — {service.service_key}</DialogTitle></DialogHeader>
        <div className="flex flex-col gap-4 py-4">
          <div className="flex flex-col gap-1.5">
            <Label>服务名称 *</Label>
            <Input value={form.service_name ?? ''} onChange={(e) => setForm((f) => ({ ...f, service_name: e.target.value }))} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>仓库地址</Label>
            <Input value={form.repo_url ?? ''} onChange={(e) => setForm((f) => ({ ...f, repo_url: e.target.value }))} />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button onClick={handleSave} disabled={submitting}>{submitting ? '保存中...' : '保存'}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}


function CreateEnvDialog({ open, onOpenChange, hosts, onCreate }: {
  open: boolean; onOpenChange: (o: boolean) => void; hosts: Host[]; onCreate: (p: CreateEnvPayload) => Promise<void>
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
      toast.success('环境创建成功'); onOpenChange(false); setForm({ env_code: 'dev' }); setLogFields({}); setJenkinsJob('')
    } catch (err) { toast.error(err instanceof Error ? err.message : '创建失败') }
    finally { setSubmitting(false) }
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg" onClose={() => onOpenChange(false)}>
        <DialogHeader><DialogTitle>新建环境</DialogTitle></DialogHeader>
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

function EditEnvDialog({ open, env, hosts, onOpenChange, onSave }: {
  open: boolean; env: ServiceEnv; hosts: Host[]; onOpenChange: (o: boolean) => void; onSave: (p: UpdateEnvPayload) => Promise<void>
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
      toast.success('环境已更新'); onOpenChange(false)
    } catch (err) { toast.error(err instanceof Error ? err.message : '保存失败') }
    finally { setSubmitting(false) }
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg" onClose={() => onOpenChange(false)}>
        <DialogHeader><DialogTitle>编辑环境 — {env.env_code}</DialogTitle></DialogHeader>
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
          {form.health_check_enabled !== false && (
            <div className="grid grid-cols-2 gap-3">
              <div className="flex flex-col gap-1.5"><Label>检查间隔 (秒)</Label><Input type="number" value={form.health_check_interval ?? 60} onChange={(e) => setForm((f) => ({ ...f, health_check_interval: Number(e.target.value) }))} /></div>
              <div className="flex flex-col gap-1.5"><Label>超时 (秒)</Label><Input type="number" value={form.health_check_timeout ?? 10} onChange={(e) => setForm((f) => ({ ...f, health_check_timeout: Number(e.target.value) }))} /></div>
            </div>
          )}
          <div className="flex flex-col gap-1.5">
            <Label>日志源类型</Label>
            <Select options={LOG_SOURCE_OPTIONS} value={form.log_source_type ?? ''} onChange={(e) => { const t = e.target.value; setForm((f) => ({ ...f, log_source_type: t })); setLogFields(parseLogSourceConfig(t, env.log_source_config ?? '{}')) }} />
          </div>
          {form.log_source_type && <LogSourceConfigEditor type_={form.log_source_type} fields={logFields} onChange={setLogFields} />}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button onClick={handleSave} disabled={submitting}>{submitting ? '保存中…' : '保存'}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
