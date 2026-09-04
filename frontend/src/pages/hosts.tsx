import { useEffect, useState, useMemo } from 'react'
import { useHostStore } from '@/stores/host-store'
import { useCredentialStore } from '@/stores/credential-store'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Plus, Trash2, Server, Wifi, WifiOff, HelpCircle, Pencil, Activity, ChevronDown, ChevronRight, ShieldAlert } from 'lucide-react'
import { toast } from 'sonner'
import { http } from '@/lib/http'
import { HostsSubTabs } from '@/components/layout/hosts-sub-tabs'
import type { AgentStatus, CreateHostPayload, UpdateHostPayload, Host } from '@/types/api'

const STATUS_MAP: Record<AgentStatus, { label: string; icon: React.ReactNode; variant: 'default' | 'secondary' | 'destructive' | 'outline' }> = {
  online: { label: '在线', icon: <Wifi className="size-3" />, variant: 'default' },
  offline: { label: '离线', icon: <WifiOff className="size-3" />, variant: 'destructive' },
  unknown: { label: '未知', icon: <HelpCircle className="size-3" />, variant: 'outline' },
}

export default function HostsPage() {
  const hosts = useHostStore((s) => s.hosts)
  const total = useHostStore((s) => s.total)
  const loading = useHostStore((s) => s.loading)
  const fetchHosts = useHostStore((s) => s.fetchHosts)
  const createHost = useHostStore((s) => s.createHost)
  const updateHost = useHostStore((s) => s.updateHost)
  const deleteHost = useHostStore((s) => s.deleteHost)
  const testConnection = useHostStore((s) => s.testConnection)

  const credentials = useCredentialStore((s) => s.credentials)
  const fetchCredentials = useCredentialStore((s) => s.fetchCredentials)

  const [showCreate, setShowCreate] = useState(false)
  const [editingHost, setEditingHost] = useState<Host | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)
  const [testingId, setTestingId] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [metricHost, setMetricHost] = useState<Host | null>(null)
  const [form, setForm] = useState<CreateHostPayload>({
    name: '',
    host_address: '',
    ssh_port: 22,
    username: '',
    credential_id: '',
    is_prod: false,
    description: '',
  })

  useEffect(() => {
    fetchHosts()
    fetchCredentials()
  }, [fetchHosts, fetchCredentials])

  const resetForm = () =>
    setForm({ name: '', host_address: '', ssh_port: 22, username: '', credential_id: '', is_prod: false, description: '' })

  const handleCreate = async () => {
    if (!form.name.trim() || !form.host_address.trim()) {
      toast.error('名称和主机地址不能为空')
      return
    }
    if (form.credential_id && !form.username?.trim()) {
      toast.error('选择凭证时 SSH 用户名不能为空')
      return
    }
    setSubmitting(true)
    try {
      const payload = { ...form, credential_id: form.credential_id || undefined }
      await createHost(payload)
      toast.success('主机已添加')
      setShowCreate(false)
      resetForm()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleTestConnection = async (id: string) => {
    setTestingId(id)
    try {
      const result = await testConnection(id)
      if (result.success) {
        toast.success(`连接成功！${result.os_info ? `\n${result.os_info}` : ''} (${result.latency_ms}ms)`)
        fetchHosts()
      } else {
        toast.error(`连接失败: ${result.error}`)
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '测试失败')
    } finally {
      setTestingId(null)
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteHost(deleteTarget)
      toast.success('主机已删除')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '删除失败')
    } finally {
      setDeleteTarget(null)
    }
  }

  const credOptions = [
    { value: '', label: '无凭证' },
    ...credentials.map((c) => ({ value: c.id, label: `${c.name} (${c.cred_type})` })),
  ]

  return (
    <div className="flex flex-col gap-4">
      <HostsSubTabs />
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">主机管理</h1>
          <p className="text-muted-foreground text-sm">共 {total} 台主机</p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus data-icon="inline-start" />
          添加主机
        </Button>
      </div>

      {loading ? (
        <div className="flex flex-col gap-2">
          {[...Array(4)].map((_, i) => <Skeleton key={i} className="h-12 w-full rounded" />)}
        </div>
      ) : (
        <div className="rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>地址</TableHead>
                <TableHead>端口</TableHead>
                <TableHead>SSH 用户</TableHead>
                <TableHead>环境</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>系统信息</TableHead>
                <TableHead>最近心跳</TableHead>
                <TableHead className="w-32">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {hosts.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={9} className="text-center text-muted-foreground py-12">
                    <Server className="mx-auto mb-2 size-8 opacity-30" />
                    暂无主机，点击右上角添加
                  </TableCell>
                </TableRow>
              ) : (
                hosts.map((h) => {
                  const statusInfo = STATUS_MAP[h.agent_status] ?? STATUS_MAP.unknown
                  return (
                    <TableRow key={h.id}>
                      <TableCell className="font-medium">{h.name}</TableCell>
                      <TableCell className="font-mono text-sm">{h.host_address}</TableCell>
                      <TableCell>{h.ssh_port}</TableCell>
                      <TableCell>{h.username || '-'}</TableCell>
                      <TableCell>
                        {h.is_prod ? (
                          <Badge variant="destructive" className="gap-1">
                            <ShieldAlert className="size-3" />
                            Prod
                          </Badge>
                        ) : (
                          <Badge variant="outline">普通</Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        <Badge variant={statusInfo.variant} className="gap-1">
                          {statusInfo.icon}
                          {statusInfo.label}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground w-40 max-w-40 truncate">{h.os_info || '-'}</TableCell>
                      <TableCell className="text-xs">
                        {h.last_heartbeat ? new Date(h.last_heartbeat).toLocaleString() : '-'}
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setMetricHost(h)}
                            title="资源详情"
                          >
                            <Activity className="size-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setEditingHost(h)}
                          >
                            <Pencil className="size-4" />
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={testingId === h.id}
                            onClick={() => handleTestConnection(h.id)}
                          >
                            {testingId === h.id ? '测试中…' : '连接测试'}
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-destructive hover:text-destructive"
                            onClick={() => setDeleteTarget(h.id)}
                          >
                            <Trash2 className="size-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })
              )}
            </TableBody>
          </Table>
        </div>
      )}

      {metricHost && (
        <HostMetricDialog
          host={metricHost}
          open={!!metricHost}
          onOpenChange={(o) => { if (!o) setMetricHost(null) }}
        />
      )}

      {editingHost && (
        <EditHostDialog
          open={!!editingHost}
          host={editingHost}
          credOptions={credOptions}
          onOpenChange={(o) => { if (!o) setEditingHost(null) }}
          onSave={async (p) => { await updateHost(editingHost.id, p); setEditingHost(null); fetchHosts() }}
        />
      )}

      {/* 新建主机 Dialog */}
      <Dialog open={showCreate} onOpenChange={(o) => { setShowCreate(o); if (!o) resetForm() }}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>添加主机</DialogTitle>
            <DialogDescription>添加一台 SSH 可达的目标主机</DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-4 py-2">
            <div className="flex flex-col gap-1.5">
              <Label>名称 *</Label>
              <Input
                placeholder="例：prod-server-01"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              />
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div className="col-span-2 flex flex-col gap-1.5">
                <Label>主机地址 *</Label>
                <Input
                  placeholder="IP 或域名"
                  value={form.host_address}
                  onChange={(e) => setForm((f) => ({ ...f, host_address: e.target.value }))}
                />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label>SSH 端口</Label>
                <Input
                  type="number"
                  value={form.ssh_port}
                  onChange={(e) => setForm((f) => ({ ...f, ssh_port: Number(e.target.value) }))}
                />
              </div>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>SSH 用户名</Label>
              <Input
                placeholder="例：root / deploy"
                value={form.username ?? ''}
                onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>关联凭证</Label>
              <Select
                value={form.credential_id ?? ''}
                onChange={(e) => setForm((f) => ({ ...f, credential_id: e.target.value }))}
                options={credOptions}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>服务器级别</Label>
              <Select
                value={form.is_prod ? 'true' : 'false'}
                onChange={(e) => setForm((f) => ({ ...f, is_prod: e.target.value === 'true' }))}
                options={[
                  { value: 'false', label: '普通服务器' },
                  { value: 'true', label: 'Prod 线上服务器' },
                ]}
              />
              <p className="text-xs text-muted-foreground">Prod 主机不能默认作为数据库迁移或对象同步的目标。</p>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>描述</Label>
              <Input
                placeholder="可选备注"
                value={form.description ?? ''}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowCreate(false); resetForm() }}>取消</Button>
            <Button onClick={handleCreate} disabled={submitting}>
              {submitting ? '添加中…' : '添加'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 删除确认 */}
      <Dialog open={!!deleteTarget} onOpenChange={(o) => { if (!o) setDeleteTarget(null) }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>此主机删除后不可恢复。</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>取消</Button>
            <Button variant="destructive" onClick={handleDelete}>确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function EditHostDialog({ open, host, credOptions, onOpenChange, onSave }: {
  open: boolean
  host: Host
  credOptions: { value: string; label: string }[]
  onOpenChange: (o: boolean) => void
  onSave: (p: UpdateHostPayload) => Promise<void>
}) {
  const [form, setForm] = useState<UpdateHostPayload>({
    name: host.name,
    host_address: host.host_address,
    ssh_port: host.ssh_port,
    username: host.username,
    credential_id: host.credential_id,
    is_prod: host.is_prod,
    description: host.description,
  })
  const [submitting, setSubmitting] = useState(false)
  const handleSave = async () => {
    if (!form.name?.trim() || !form.host_address?.trim()) { toast.error('名称和主机地址不能为空'); return }
    if (form.credential_id && !form.username?.trim()) { toast.error('选择凭证时 SSH 用户名不能为空'); return }
    setSubmitting(true)
    try { await onSave({ ...form, credential_id: form.credential_id || undefined }); toast.success('主机已更新') }
    catch (err) { toast.error(err instanceof Error ? err.message : '更新失败') }
    finally { setSubmitting(false) }
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>编辑主机</DialogTitle>
          <DialogDescription>修改主机连接信息</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4 py-2">
          <div className="flex flex-col gap-1.5">
            <Label>名称 *</Label>
            <Input value={form.name ?? ''} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div className="col-span-2 flex flex-col gap-1.5">
              <Label>主机地址 *</Label>
              <Input value={form.host_address ?? ''} onChange={(e) => setForm((f) => ({ ...f, host_address: e.target.value }))} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>SSH 端口</Label>
              <Input type="number" value={form.ssh_port ?? 22} onChange={(e) => setForm((f) => ({ ...f, ssh_port: Number(e.target.value) }))} />
            </div>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>SSH 用户名</Label>
            <Input
              placeholder="例：root / deploy"
              value={form.username ?? ''}
              onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>关联凭证</Label>
            <Select value={form.credential_id ?? ''} onChange={(e) => setForm((f) => ({ ...f, credential_id: e.target.value }))} options={credOptions} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>服务器级别</Label>
            <Select
              value={form.is_prod ? 'true' : 'false'}
              onChange={(e) => setForm((f) => ({ ...f, is_prod: e.target.value === 'true' }))}
              options={[
                { value: 'false', label: '普通服务器' },
                { value: 'true', label: 'Prod 线上服务器' },
              ]}
            />
            <p className="text-xs text-muted-foreground">Prod 主机会触发迁移/同步目标保护。</p>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>描述</Label>
            <Input value={form.description ?? ''} onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))} />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button onClick={handleSave} disabled={submitting}>{submitting ? '保存中…' : '保存'}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ── 主机资源详情弹窗 ──────────────────────────────────

interface HostMetric {
  cpu_usage: number
  cpu_cores: number
  mem_total_mb: number
  mem_used_mb: number
  mem_usage: number
  disk_total_gb: number
  disk_used_gb: number
  disk_usage: number
  disks_json: string
  load_avg_1: number
  load_avg_5: number
  load_avg_15: number
  net_in_bytes: number
  net_out_bytes: number
  gpu_usage: number | null
  gpu_mem_usage: number | null
  gpu_temp: number | null
  gpu_name: string
  gpus?: Array<{ index: number; name: string; usage: number; mem_usage: number; temp: number }> | null
  collected_at: string
}

interface DiskInfo {
  mount_point: string
  total_gb: number
  used_gb: number
  usage: number
}

function usageColor(v: number) {
  if (v >= 90) return 'text-red-600'
  if (v >= 70) return 'text-yellow-600'
  return 'text-green-600'
}

function usageBg(v: number) {
  if (v >= 90) return 'bg-red-500'
  if (v >= 70) return 'bg-yellow-500'
  return 'bg-green-500'
}

function ProgressBar({ value }: { value: number }) {
  return (
    <div className="h-2 w-full rounded-full bg-muted">
      <div className={`h-full rounded-full transition-all ${usageBg(value)}`} style={{ width: `${Math.min(value, 100)}%` }} />
    </div>
  )
}

function MetricRow({ label, value, extra }: { label: string; value: number; extra?: string }) {
  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">{label}</span>
        <span className={`font-medium ${usageColor(value)}`}>
          {value.toFixed(1)}%
          {extra && <span className="text-muted-foreground ml-1 font-normal text-xs">({extra})</span>}
        </span>
      </div>
      <ProgressBar value={value} />
    </div>
  )
}

function HostMetricDialog({ host, open, onOpenChange }: {
  host: Host
  open: boolean
  onOpenChange: (o: boolean) => void
}) {
  const [metric, setMetric] = useState<HostMetric | null>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open) return
    setLoading(true)
    http.get<HostMetric[]>('/api/v1/hosts/metrics/overview')
      .then((res) => {
        const m = (res.data ?? []).find((item: any) => item.host_id === host.id)
        setMetric(m ?? null)
      })
      .catch(() => setMetric(null))
      .finally(() => setLoading(false))
  }, [open, host.id])

  const disks: DiskInfo[] = useMemo(() => {
    if (!metric?.disks_json) return []
    try { return JSON.parse(metric.disks_json) } catch { return [] }
  }, [metric?.disks_json])

  const gpus = metric?.gpus ?? []
  const [diskExpanded, setDiskExpanded] = useState(false)
  const [gpuExpanded, setGpuExpanded] = useState(false)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Activity className="size-4" />
            {host.name} 资源详情
          </DialogTitle>
          <DialogDescription className="font-mono text-xs">{host.host_address} {host.os_info ? `· ${host.os_info}` : ''}</DialogDescription>
        </DialogHeader>

        {loading ? (
          <div className="flex flex-col gap-3 py-4">
            {[1,2,3].map(i => <Skeleton key={i} className="h-8 w-full" />)}
          </div>
        ) : !metric ? (
          <p className="text-sm text-muted-foreground py-6 text-center">暂无指标数据（主机可能离线或 Agent 未部署）</p>
        ) : (
          <div className="flex flex-col gap-4 py-2">
            <MetricRow label="CPU" value={metric.cpu_usage} extra={`${metric.cpu_cores} 核`} />
            <MetricRow label="内存" value={metric.mem_usage} extra={`${metric.mem_used_mb}/${metric.mem_total_mb} MB`} />
            <MetricRow label="磁盘（总计）" value={metric.disk_usage} extra={`${metric.disk_used_gb.toFixed(1)}/${metric.disk_total_gb.toFixed(1)} GB`} />

            {/* 负载 */}
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">系统负载</span>
              <span className="font-mono text-xs">
                {metric.load_avg_1.toFixed(2)} / {metric.load_avg_5.toFixed(2)} / {metric.load_avg_15.toFixed(2)}
              </span>
            </div>

            {/* 网络 */}
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">网络 I/O</span>
              <span className="font-mono text-xs">
                ↓ {(metric.net_in_bytes / 1024).toFixed(1)} KB/s &nbsp; ↑ {(metric.net_out_bytes / 1024).toFixed(1)} KB/s
              </span>
            </div>

            {/* 磁盘分区 */}
            {disks.length > 0 && (
              <div>
                <button
                  type="button"
                  className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
                  onClick={() => setDiskExpanded(v => !v)}
                >
                  {diskExpanded ? <ChevronDown className="size-3.5" /> : <ChevronRight className="size-3.5" />}
                  磁盘分区（{disks.length}）
                </button>
                {diskExpanded && (
                  <div className="mt-2 flex flex-col gap-2 pl-2 border-l-2 border-muted">
                    {disks.map(d => (
                      <div key={d.mount_point} className="flex flex-col gap-0.5">
                        <div className="flex items-center justify-between text-xs">
                          <span className="text-muted-foreground font-mono">{d.mount_point}</span>
                          <span className={`font-medium ${usageColor(d.usage)}`}>
                            {d.usage.toFixed(1)}% <span className="text-muted-foreground font-normal">({d.used_gb.toFixed(1)}/{d.total_gb.toFixed(1)} GB)</span>
                          </span>
                        </div>
                        <ProgressBar value={d.usage} />
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* GPU */}
            {gpus.length > 0 && (
              <div>
                <button
                  type="button"
                  className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
                  onClick={() => setGpuExpanded(v => !v)}
                >
                  {gpuExpanded ? <ChevronDown className="size-3.5" /> : <ChevronRight className="size-3.5" />}
                  GPU × {gpus.length}
                </button>
                {gpuExpanded && (
                  <div className="mt-2 flex flex-col gap-2 pl-2 border-l-2 border-muted">
                    {gpus.map(g => (
                      <div key={g.index} className="rounded-md border bg-muted/20 px-3 py-2 flex flex-col gap-1">
                        <div className="flex items-center justify-between text-xs">
                          <span className="font-mono text-muted-foreground">#{g.index}</span>
                          <span className="truncate text-muted-foreground max-w-[60%]" title={g.name}>{g.name}</span>
                        </div>
                        <MetricRow label="使用率" value={g.usage} />
                        <MetricRow label="显存" value={g.mem_usage} />
                        <div className="flex items-center justify-between text-xs">
                          <span className="text-muted-foreground">温度</span>
                          <span className={`font-mono ${g.temp > 80 ? 'text-red-600' : g.temp > 70 ? 'text-yellow-600' : ''}`}>
                            {g.temp.toFixed(0)}°C
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
            {!gpus.length && metric.gpu_usage !== null && (
              <>
                <MetricRow label="GPU" value={metric.gpu_usage ?? 0} extra={metric.gpu_name || undefined} />
                {metric.gpu_temp !== null && (
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">GPU 温度</span>
                    <span className={`font-mono text-xs ${(metric.gpu_temp ?? 0) > 80 ? 'text-red-600' : ''}`}>{metric.gpu_temp}°C</span>
                  </div>
                )}
              </>
            )}

            <p className="text-xs text-muted-foreground text-right">
              采集于 {metric.collected_at ? new Date(metric.collected_at).toLocaleString() : '-'}
            </p>
          </div>
        )}
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>关闭</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
