import { useEffect, useState, useCallback, useMemo } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell,
} from '@/components/ui/table'
import { http } from '@/lib/http'
import { useHostStore } from '@/stores/host-store'
import {
  RefreshCw,
  Search, LayoutGrid, TableProperties, ChevronDown, ChevronRight,
} from 'lucide-react'
import { HostsSubTabs } from '@/components/layout/hosts-sub-tabs'
import type { Host } from '@/types/api'

// ── Types ──────────────────────────────────────────────

interface DiskInfo {
  mount_point: string
  total_gb: number
  used_gb: number
  usage: number
}

interface HostMetric {
  id: string
  host_id: string
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
  gpus?: GPUInfo[] | null
  collected_at: string
}

interface GPUInfo {
  index: number
  name: string
  usage: number
  mem_usage: number
  temp: number
}

type SortKey = 'name' | 'cpu' | 'mem' | 'disk'
type SortDir = 'asc' | 'desc'
type ViewMode = 'card' | 'table'

// ── Helpers ────────────────────────────────────────────

function usageColor(value: number): string {
  if (value >= 90) return 'text-red-600'
  if (value >= 70) return 'text-yellow-600'
  return 'text-green-600'
}

function usageBg(value: number): string {
  if (value >= 90) return 'bg-red-500'
  if (value >= 70) return 'bg-yellow-500'
  return 'bg-green-500'
}

function parseDisks(json: string | undefined | null): DiskInfo[] {
  if (!json) return []
  try {
    const arr = JSON.parse(json)
    return Array.isArray(arr) ? arr : []
  } catch {
    return []
  }
}

function formatTime(s: string) {
  if (!s) return '-'
  const d = new Date(s)
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

// ── Shared UI pieces ──────────────────────────────────

function ProgressBar({ value, className }: { value: number; className?: string }) {
  return (
    <div className={`h-2 w-full rounded-full bg-muted ${className || ''}`}>
      <div
        className={`h-full rounded-full transition-all ${usageBg(value)}`}
        style={{ width: `${Math.min(value, 100)}%` }}
      />
    </div>
  )
}

function MetricRow({ label, value, extra }: { label: string; value: number; extra?: string }) {
  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center justify-between text-xs">
        <span className="text-muted-foreground">{label}</span>
        <span className={`font-medium ${usageColor(value)}`}>
          {value.toFixed(1)}%
          {extra && <span className="text-muted-foreground ml-1 font-normal">({extra})</span>}
        </span>
      </div>
      <ProgressBar value={value} />
    </div>
  )
}



// ── Disk sub-rows ─────────────────────────────────────

function DiskPartitions({ disks }: { disks: DiskInfo[] }) {
  const [expanded, setExpanded] = useState(false)
  if (disks.length === 0) return null

  return (
    <div className="mt-1">
      <button
        type="button"
        className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
        onClick={() => setExpanded((v) => !v)}
      >
        {expanded ? <ChevronDown className="size-3" /> : <ChevronRight className="size-3" />}
        磁盘分区（{disks.length}）
      </button>
      {expanded && (
        <div className="mt-1.5 flex flex-col gap-2 pl-1 border-l-2 border-muted">
          {disks.map((d) => (
            <div key={d.mount_point} className="flex flex-col gap-0.5">
              <div className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground font-mono truncate max-w-[120px]" title={d.mount_point}>
                  {d.mount_point}
                </span>
                <span className={`font-medium ${usageColor(d.usage)}`}>
                  {d.usage.toFixed(1)}%
                  <span className="text-muted-foreground ml-1 font-normal">
                    ({d.used_gb.toFixed(1)}/{d.total_gb.toFixed(1)} GB)
                  </span>
                </span>
              </div>
              <ProgressBar value={d.usage} />
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ── Card View ─────────────────────────────────────────

// ── GPU section (with collapse) ───────────────────────

function GPUSection({ metric }: { metric: HostMetric }) {
  const gpus = metric.gpus ?? []
  const hasMulti = gpus.length > 0
  const hasLegacy = !hasMulti && metric.gpu_usage !== null
  // 默认折叠
  const [expanded, setExpanded] = useState(false)

  if (!hasMulti && !hasLegacy) return null

  if (!hasMulti) {
    // 旧字段单卡渲染
    return (
      <div className="mt-1 pt-2 border-t border-border/60 flex flex-col gap-1.5">
        {metric.gpu_usage !== null && (
          <MetricRow label="GPU" value={metric.gpu_usage ?? 0} extra={metric.gpu_name || undefined} />
        )}
        {metric.gpu_temp !== null && (
          <div className="flex items-center justify-between text-xs">
            <span className="text-muted-foreground">GPU 温度</span>
            <span className={`font-mono ${(metric.gpu_temp ?? 0) > 80 ? 'text-red-600' : ''}`}>
              {metric.gpu_temp}°C
            </span>
          </div>
        )}
      </div>
    )
  }

  // 多卡渲染（可折叠）
  const maxUsage = Math.max(...gpus.map(g => g.usage))
  const maxTemp = Math.max(...gpus.map(g => g.temp))
  return (
    <div className="mt-1 pt-2 border-t border-border/60">
      <button
        type="button"
        className="flex items-center justify-between w-full text-xs hover:text-foreground transition-colors"
        onClick={() => setExpanded(v => !v)}
      >
        <span className="flex items-center gap-1 text-muted-foreground">
          {expanded ? <ChevronDown className="size-3" /> : <ChevronRight className="size-3" />}
          GPU × {gpus.length}
        </span>
        <span className="flex items-center gap-2 text-[11px]">
          <span className={`font-mono ${usageColor(maxUsage)}`}>峰值 {maxUsage.toFixed(0)}%</span>
          <span className={`font-mono ${maxTemp > 80 ? 'text-red-600' : maxTemp > 70 ? 'text-yellow-600' : 'text-muted-foreground'}`}>
            {maxTemp.toFixed(0)}°C
          </span>
        </span>
      </button>
      {expanded && (
        <div className="mt-1.5 flex flex-col gap-1.5 pl-1 border-l-2 border-muted">
          {gpus.map((g) => (
            <div key={g.index} className="flex flex-col gap-1 rounded-md border border-border/60 bg-muted/20 px-2 py-1.5">
              <div className="flex items-center justify-between text-xs">
                <span className="font-mono text-[11px] text-muted-foreground">#{g.index}</span>
                <span className="truncate text-[11px] text-muted-foreground max-w-[60%]" title={g.name}>{g.name}</span>
              </div>
              <MetricRow label="使用率" value={g.usage} />
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
  )
}

// ── Card content ──────────────────────────────────────

function HostMetricCard({ host, metric }: { host: Host; metric?: HostMetric }) {
  const disks = metric ? parseDisks(metric.disks_json) : []

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium truncate" title={host.name}>{host.name}</CardTitle>
          <Badge variant={host.agent_status === 'online' ? 'default' : 'secondary'}>
            {host.agent_status === 'online' ? '在线' : host.agent_status === 'offline' ? '离线' : '未知'}
          </Badge>
        </div>
        <p className="text-xs text-muted-foreground font-mono">{host.host_address}</p>
      </CardHeader>
      <CardContent>
        {!metric ? (
          <p className="text-xs text-muted-foreground">暂无指标数据</p>
        ) : (
          <div className="flex flex-col gap-3">
            <MetricRow label="CPU" value={metric.cpu_usage} extra={`${metric.cpu_cores} 核`} />
            <MetricRow
              label="内存"
              value={metric.mem_usage}
              extra={`${metric.mem_used_mb}/${metric.mem_total_mb} MB`}
            />
            {disks.length > 0 ? (
              <DiskPartitions disks={disks} />
            ) : (
              <MetricRow label="磁盘" value={metric.disk_usage} />
            )}
            <GPUSection metric={metric} />
            <p className="text-xs text-muted-foreground text-right">
              采集于 {formatTime(metric.collected_at)}
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// ── Table View ────────────────────────────────────────

function HostTable({ hosts, metrics }: { hosts: Host[]; metrics: Record<string, HostMetric> }) {
  return (
    <Card>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>名称</TableHead>
            <TableHead>IP</TableHead>
            <TableHead>状态</TableHead>
            <TableHead className="text-right">CPU%</TableHead>
            <TableHead className="text-right">内存%</TableHead>
            <TableHead className="text-right">磁盘%</TableHead>
            <TableHead>磁盘分区</TableHead>
            <TableHead className="text-right">负载</TableHead>
            <TableHead className="text-right">GPU</TableHead>
            <TableHead className="text-right">更新时间</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {hosts.length === 0 ? (
            <TableRow>
              <TableCell colSpan={9} className="text-center text-muted-foreground py-8">
                无匹配主机
              </TableCell>
            </TableRow>
          ) : (
            hosts.map((h) => {
              const m = metrics[h.id]
              return (
                <TableRow key={h.id}>
                  <TableCell className="font-medium truncate max-w-[140px]" title={h.name}>{h.name}</TableCell>
                  <TableCell className="font-mono text-xs">{h.host_address}</TableCell>
                  <TableCell>
                    <Badge variant={h.agent_status === 'online' ? 'default' : 'secondary'} className="text-xs">
                      {h.agent_status === 'online' ? '在线' : h.agent_status === 'offline' ? '离线' : '未知'}
                    </Badge>
                  </TableCell>
                  {m ? (
                    <>
                      <TableCell className={`text-right font-mono ${usageColor(m.cpu_usage)}`}>
                        {m.cpu_usage.toFixed(1)}
                      </TableCell>
                      <TableCell className={`text-right font-mono ${usageColor(m.mem_usage)}`}>
                        {m.mem_usage.toFixed(1)}
                      </TableCell>
                      <TableCell className={`text-right font-mono ${usageColor(m.disk_usage)}`}>
                        {m.disk_usage.toFixed(1)}
                      </TableCell>
                      <TableCell className="text-xs">
                        {(() => {
                          const disks = parseDisks(m.disks_json)
                          if (disks.length === 0) return <span className="text-muted-foreground">-</span>
                          return (
                            <div className="flex flex-col gap-0.5">
                              {disks.map((d) => (
                                <span key={d.mount_point} className="font-mono">
                                  <span className="text-muted-foreground">{d.mount_point}</span>{' '}
                                  <span className={usageColor(d.usage)}>{d.usage.toFixed(0)}%</span>{' '}
                                  <span className="text-muted-foreground">({d.used_gb.toFixed(0)}/{d.total_gb.toFixed(0)}G)</span>
                                </span>
                              ))}
                            </div>
                          )
                        })()}
                      </TableCell>
                      <TableCell className="text-right font-mono text-xs">
                        {m.load_avg_5.toFixed(2)}
                      </TableCell>
                      <TableCell className="text-right font-mono text-xs">
                        {m.gpus && m.gpus.length > 0
                          ? `${m.gpus.length}卡 · 最高 ${Math.max(...m.gpus.map(g => g.usage)).toFixed(0)}%`
                          : m.gpu_usage !== null ? `${(m.gpu_usage).toFixed(0)}%` : '-'}
                      </TableCell>
                      <TableCell className="text-right text-xs text-muted-foreground">
                        {formatTime(m.collected_at)}
                      </TableCell>
                    </>
                  ) : (
                    <TableCell colSpan={6} className="text-center text-xs text-muted-foreground">
                      暂无数据
                    </TableCell>
                  )}
                </TableRow>
              )
            })
          )}
        </TableBody>
      </Table>
    </Card>
  )
}

// ── Sort options ──────────────────────────────────────

const SORT_OPTIONS: { value: string; label: string }[] = [
  { value: 'name-asc', label: '名称 A→Z' },
  { value: 'name-desc', label: '名称 Z→A' },
  { value: 'cpu-desc', label: 'CPU ↓' },
  { value: 'cpu-asc', label: 'CPU ↑' },
  { value: 'mem-desc', label: '内存 ↓' },
  { value: 'mem-asc', label: '内存 ↑' },
  { value: 'disk-desc', label: '磁盘 ↓' },
  { value: 'disk-asc', label: '磁盘 ↑' },
]

function parseSortValue(v: string): { key: SortKey; dir: SortDir } {
  const [key, dir] = v.split('-') as [SortKey, SortDir]
  return { key, dir }
}

// ── Main Page ─────────────────────────────────────────

export default function MonitorPage() {
  const hosts = useHostStore((s) => s.hosts)
  const fetchHosts = useHostStore((s) => s.fetchHosts)
  const [metrics, setMetrics] = useState<Record<string, HostMetric>>({})
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  const [search, setSearch] = useState('')
  const [sortValue, setSortValue] = useState('cpu-desc')
  const [viewMode, setViewMode] = useState<ViewMode>('card')

  useEffect(() => {
    fetchHosts()
  }, [fetchHosts])

  const loadMetrics = useCallback(async () => {
    try {
      const res = await http.get<HostMetric[]>('/api/v1/hosts/metrics/overview')
      const data = Array.isArray(res.data) ? res.data : []
      const map: Record<string, HostMetric> = {}
      for (const m of data) {
        map[m.host_id] = m
      }
      setMetrics(map)
    } catch { /* ignore */ }
    setLoading(false)
  }, [])

  useEffect(() => {
    loadMetrics()
    const timer = setInterval(loadMetrics, 30000)
    return () => clearInterval(timer)
  }, [loadMetrics])

  const handleRefresh = async () => {
    setRefreshing(true)
    await loadMetrics()
    setRefreshing(false)
  }

  // ── Derived data ──

  const filteredAndSorted = useMemo(() => {
    const q = search.trim().toLowerCase()
    let list = hosts.filter((h) => {
      if (!q) return true
      return h.name.toLowerCase().includes(q) || h.host_address.toLowerCase().includes(q)
    })

    const { key, dir } = parseSortValue(sortValue)
    list = [...list].sort((a, b) => {
      const ma = metrics[a.id]
      const mb = metrics[b.id]
      let diff = 0
      switch (key) {
        case 'name':
          diff = a.name.localeCompare(b.name, 'zh-CN')
          break
        case 'cpu':
          diff = (ma?.cpu_usage ?? -1) - (mb?.cpu_usage ?? -1)
          break
        case 'mem':
          diff = (ma?.mem_usage ?? -1) - (mb?.mem_usage ?? -1)
          break
        case 'disk':
          diff = (ma?.disk_usage ?? -1) - (mb?.disk_usage ?? -1)
          break
      }
      return dir === 'asc' ? diff : -diff
    })

    return list
  }, [hosts, metrics, search, sortValue])

  // ── Render ──

  if (loading) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-8 w-48" />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {[1, 2, 3, 4].map((i) => <Skeleton key={i} className="h-24 w-full" />)}
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => <Skeleton key={i} className="h-48 w-full" />)}
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <HostsSubTabs />
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">资源监控</h1>
          <p className="text-muted-foreground text-sm">
            {hosts.length} 台主机 · {Object.keys(metrics).length} 台有指标数据 · 每 30 秒自动刷新
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={handleRefresh} disabled={refreshing}>
          <RefreshCw className={`size-4 mr-1 ${refreshing ? 'animate-spin' : ''}`} />
          刷新
        </Button>
      </div>

      {/* Toolbar: Search + Sort + View toggle */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h2 className="text-lg font-semibold shrink-0">全部主机</h2>
        <div className="flex items-center gap-2 flex-wrap">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
            <Input
              placeholder="搜索名称或 IP…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-8 w-48"
            />
          </div>
          <Select
            options={SORT_OPTIONS}
            value={sortValue}
            onChange={(e) => setSortValue(e.target.value)}
            className="w-32"
            aria-label="排序方式"
          />
          <div className="flex rounded-md border border-input overflow-hidden">
            <button
              type="button"
              className={`flex items-center gap-1 px-2.5 py-1.5 text-xs transition-colors ${
                viewMode === 'card'
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-transparent text-muted-foreground hover:bg-muted'
              }`}
              onClick={() => setViewMode('card')}
              aria-label="卡片视图"
            >
              <LayoutGrid className="size-3.5" />
              卡片
            </button>
            <button
              type="button"
              className={`flex items-center gap-1 px-2.5 py-1.5 text-xs transition-colors border-l border-input ${
                viewMode === 'table'
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-transparent text-muted-foreground hover:bg-muted'
              }`}
              onClick={() => setViewMode('table')}
              aria-label="表格视图"
            >
              <TableProperties className="size-3.5" />
              表格
            </button>
          </div>
        </div>
      </div>

      {/* Host list */}
      {viewMode === 'card' ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filteredAndSorted.length === 0 ? (
            <p className="text-sm text-muted-foreground col-span-full text-center py-12">
              {hosts.length === 0 ? '暂无主机，请在主机管理中添加' : '无匹配主机'}
            </p>
          ) : (
            filteredAndSorted.map((h) => (
              <HostMetricCard key={h.id} host={h} metric={metrics[h.id]} />
            ))
          )}
        </div>
      ) : (
        <HostTable hosts={filteredAndSorted} metrics={metrics} />
      )}
    </div>
  )
}
