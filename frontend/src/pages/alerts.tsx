import { useEffect, useState, useCallback } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { alertApi, type AlertEvent, type AlertStats } from '@/api/alerts'
import { useServiceStore } from '@/stores/service-store'
import { toast } from 'sonner'
import { useNavigate } from 'react-router-dom'
import {
  AlertTriangle,
  Bell,
  CheckCircle,
  Eye,
  Plus,
  RefreshCw,
  XCircle,
} from 'lucide-react'

const STATUS_OPTIONS = [
  { value: '', label: '全部状态' },
  { value: 'open', label: '待处理' },
  { value: 'acked', label: '已确认' },
  { value: 'closed', label: '已关闭' },
]

const SEVERITY_OPTIONS = [
  { value: '', label: '全部级别' },
  { value: 'P1', label: 'P1 紧急' },
  { value: 'P2', label: 'P2 重要' },
  { value: 'P3', label: 'P3 一般' },
  { value: 'P4', label: 'P4 提示' },
]

const SEVERITY_BADGE: Record<string, { variant: 'destructive' | 'default' | 'secondary' | 'outline'; label: string }> = {
  P1: { variant: 'destructive', label: 'P1 紧急' },
  P2: { variant: 'default', label: 'P2 重要' },
  P3: { variant: 'secondary', label: 'P3 一般' },
  P4: { variant: 'outline', label: 'P4 提示' },
}

const STATUS_BADGE: Record<string, { variant: 'destructive' | 'default' | 'secondary' | 'outline'; label: string }> = {
  open: { variant: 'destructive', label: '待处理' },
  acked: { variant: 'default', label: '已确认' },
  closed: { variant: 'secondary', label: '已关闭' },
  suppressed: { variant: 'outline', label: '已静默' },
}

export default function AlertsPage() {
  const services = useServiceStore((s) => s.services)
  const fetchServices = useServiceStore((s) => s.fetchServices)
  const navigate = useNavigate()
  const serviceNameMap = new Map(services.map((s) => [s.id, s.service_name]))

  const [alerts, setAlerts] = useState<AlertEvent[]>([])
  const [stats, setStats] = useState<AlertStats | null>(null)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [filterStatus, setFilterStatus] = useState('')
  const [filterSeverity, setFilterSeverity] = useState('')
  const [keyword, setKeyword] = useState('')
  const [createOpen, setCreateOpen] = useState(false)

  const loadAlerts = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, string | number> = { page, page_size: 20 }
      if (filterStatus) params.status = filterStatus
      if (filterSeverity) params.severity = filterSeverity
      if (keyword) params.keyword = keyword
      const res = await alertApi.list(params)
      setAlerts(res.data?.list || [])
      setTotal(res.data?.total || 0)
    } catch {
      /* ignore */
    }
    setLoading(false)
  }, [page, filterStatus, filterSeverity, keyword])

  const loadStats = useCallback(async () => {
    try {
      const res = await alertApi.stats()
      setStats(res.data)
    } catch {
      /* ignore */
    }
  }, [])

  useEffect(() => {
    loadAlerts()
    loadStats()
    fetchServices()
  }, [loadAlerts, loadStats, fetchServices])

  const handleAck = async (id: string) => {
    try {
      await alertApi.ack(id)
      toast.success('告警已确认')
      loadAlerts()
      loadStats()
    } catch {
      toast.error('操作失败')
    }
  }

  const handleClose = async (id: string) => {
    try {
      await alertApi.close(id)
      toast.success('告警已关闭')
      loadAlerts()
      loadStats()
    } catch {
      toast.error('操作失败')
    }
  }

  const handleCreate = async (data: { severity: string; title: string; content: string }) => {
    try {
      await alertApi.create(data)
      toast.success('告警已创建')
      setCreateOpen(false)
      loadAlerts()
      loadStats()
    } catch {
      toast.error('创建失败')
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">告警中心</h1>
          <p className="text-muted-foreground text-sm">共 {total} 条告警</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={() => { loadAlerts(); loadStats() }}>
            <RefreshCw className="size-4 mr-1" /> 刷新
          </Button>
          <Button size="sm" onClick={() => setCreateOpen(true)}>
            <Plus className="size-4 mr-1" /> 新建告警
          </Button>
          <Dialog open={createOpen} onOpenChange={setCreateOpen}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>新建告警</DialogTitle>
              </DialogHeader>
              <CreateAlertForm onSubmit={handleCreate} />
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* 统计卡片 */}
      {stats && (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4 lg:grid-cols-5">
          <StatCard icon={<Bell className="size-4 text-red-600" />} label="待处理" value={stats.total_open} highlight />
          <StatCard icon={<Eye className="size-4 text-blue-600" />} label="已确认" value={stats.total_acked} />
          <StatCard icon={<CheckCircle className="size-4 text-green-600" />} label="已关闭" value={stats.total_closed} />
          <StatCard icon={<AlertTriangle className="size-4 text-red-600" />} label="P1 未处理" value={stats.p1_open} highlight={stats.p1_open > 0} />
          <StatCard icon={<AlertTriangle className="size-4 text-yellow-600" />} label="P2 未处理" value={stats.p2_open} highlight={stats.p2_open > 0} />
        </div>
      )}

      {/* 过滤器 */}
      <div className="flex flex-wrap gap-3 items-end">
        <div className="w-48">
          <Label className="text-xs text-muted-foreground">状态</Label>
          <Select
            options={STATUS_OPTIONS}
            value={filterStatus}
            onChange={(e) => { setFilterStatus(e.target.value); setPage(1) }}
          />
        </div>
        <div className="w-48">
          <Label className="text-xs text-muted-foreground">级别</Label>
          <Select
            options={SEVERITY_OPTIONS}
            value={filterSeverity}
            onChange={(e) => { setFilterSeverity(e.target.value); setPage(1) }}
          />
        </div>
        <div className="flex-1 min-w-[200px]">
          <Label className="text-xs text-muted-foreground">搜索</Label>
          <Input
            placeholder="搜索告警标题..."
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') { setPage(1); loadAlerts() } }}
          />
        </div>
      </div>

      {/* 告警列表 */}
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">级别</TableHead>
                <TableHead>标题</TableHead>
                <TableHead className="w-28">关联服务</TableHead>
                <TableHead className="w-20">状态</TableHead>
                <TableHead className="w-20">来源</TableHead>
                <TableHead className="w-40">最近触发</TableHead>
                <TableHead className="w-32">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                    加载中...
                  </TableCell>
                </TableRow>
              ) : alerts.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                    暂无告警记录
                  </TableCell>
                </TableRow>
              ) : (
                alerts.map((a) => (
                  <TableRow key={a.id}>
                    <TableCell>
                      <Badge variant={SEVERITY_BADGE[a.severity]?.variant || 'secondary'}>
                        {SEVERITY_BADGE[a.severity]?.label || a.severity}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="font-medium text-sm">{a.title}</div>
                      {a.content && (
                        <div className="text-xs text-muted-foreground line-clamp-1 mt-0.5">{a.content}</div>
                      )}
                    </TableCell>
                    <TableCell>
                      {a.service_id ? (
                        <span
                          className="text-sm text-primary cursor-pointer hover:underline"
                          onClick={() => navigate(`/services/${a.service_id}`)}
                        >
                          {serviceNameMap.get(a.service_id) || a.service_id.slice(0, 8)}
                        </span>
                      ) : (
                        <span className="text-xs text-muted-foreground">-</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <Badge variant={STATUS_BADGE[a.status]?.variant || 'outline'}>
                        {STATUS_BADGE[a.status]?.label || a.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">{a.alert_source}</TableCell>
                    <TableCell className="text-xs text-muted-foreground font-mono">{a.last_seen_at}</TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        {a.status === 'open' && (
                          <Button variant="ghost" size="sm" onClick={() => handleAck(a.id)}>
                            <Eye className="size-3 mr-1" /> 确认
                          </Button>
                        )}
                        {(a.status === 'open' || a.status === 'acked') && (
                          <Button variant="ghost" size="sm" onClick={() => handleClose(a.id)}>
                            <XCircle className="size-3 mr-1" /> 关闭
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* 分页 */}
      {total > 20 && (
        <div className="flex justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>
            上一页
          </Button>
          <span className="flex items-center text-sm text-muted-foreground">
            第 {page} 页 / 共 {Math.ceil(total / 20)} 页
          </span>
          <Button variant="outline" size="sm" disabled={page * 20 >= total} onClick={() => setPage(page + 1)}>
            下一页
          </Button>
        </div>
      )}
    </div>
  )
}

function StatCard({ icon, label, value, highlight }: {
  icon: React.ReactNode; label: string; value: number; highlight?: boolean
}) {
  return (
    <Card className={highlight ? 'border-red-200/50' : ''}>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{label}</CardTitle>
        {icon}
      </CardHeader>
      <CardContent>
        <div className={`text-2xl font-bold ${highlight ? 'text-red-600' : ''}`}>{value}</div>
      </CardContent>
    </Card>
  )
}

function CreateAlertForm({ onSubmit }: { onSubmit: (data: { severity: string; title: string; content: string }) => void }) {
  const [severity, setSeverity] = useState('P3')
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')

  return (
    <div className="flex flex-col gap-4">
      <div>
        <Label>级别</Label>
        <Select
          options={SEVERITY_OPTIONS.filter(o => o.value !== '')}
          value={severity}
          onChange={(e) => setSeverity(e.target.value)}
        />
      </div>
      <div>
        <Label>标题</Label>
        <Input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="告警标题" />
      </div>
      <div>
        <Label>详情</Label>
        <Textarea value={content} onChange={(e) => setContent(e.target.value)} placeholder="告警详情描述（可选）" rows={3} />
      </div>
      <Button onClick={() => onSubmit({ severity, title, content })} disabled={!title.trim()}>
        创建告警
      </Button>
    </div>
  )
}
