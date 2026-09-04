import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useServiceStore } from '@/stores/service-store'
import { useReleaseStore } from '@/stores/release-store'
import { alertApi, type AlertStats } from '@/api/alerts'
import { http } from '@/lib/http'
import { Server, Rocket, CheckCircle, AlertTriangle, Activity, ArrowRight, Bell, Loader2 } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import type { ServiceEnv } from '@/types/api'

export default function DashboardPage() {
  const services = useServiceStore((s) => s.services)
  const serviceTotal = useServiceStore((s) => s.total)
  const fetchServices = useServiceStore((s) => s.fetchServices)
  const releases = useReleaseStore((s) => s.releases)
  const fetchReleases = useReleaseStore((s) => s.fetchReleases)
  const navigate = useNavigate()

  const [allEnvs, setAllEnvs] = useState<(ServiceEnv & { service_name?: string })[]>([])
  const [envsLoaded, setEnvsLoaded] = useState(false)
  const [alertStats, setAlertStats] = useState<AlertStats | null>(null)

  useEffect(() => {
    fetchServices()
    fetchReleases()
    alertApi.stats().then((res) => setAlertStats(res.data)).catch(() => {})
  }, [fetchServices, fetchReleases])

  // 加载所有服务的环境以展示健康状态汇总
  useEffect(() => {
    if (services.length > 0 && !envsLoaded) {
      const loadEnvs = async () => {
        const results: (ServiceEnv & { service_name?: string })[] = []
        for (const svc of services.slice(0, 20)) {
          try {
            const res = await http.get<ServiceEnv[]>(`/api/v1/services/${svc.id}/envs`)
            const envList = Array.isArray(res.data) ? res.data : []
            for (const env of envList) {
              results.push({ ...env, service_name: svc.service_name })
            }
          } catch { /* skip */ }
        }
        setAllEnvs(results)
        setEnvsLoaded(true)
      }
      loadEnvs()
    }
  }, [services, envsLoaded])

  // 构建服务名查找表
  const serviceNameMap = new Map(services.map((s) => [s.id, s.service_name]))

  // 环境健康统计
  const healthyCount = allEnvs.filter((e) => e.health_status === 'healthy').length
  const unhealthyEnvs = allEnvs.filter((e) => e.health_status === 'unhealthy' || e.health_status === 'unreachable')
  const unknownCount = allEnvs.filter((e) => e.health_status === 'unknown' || !e.health_status).length

  // 今日发布统计
  const today = new Date().toISOString().slice(0, 10)
  const todayReleases = releases.filter((r) => r.created_at?.startsWith(today))
  const todaySuccess = todayReleases.filter((r) => r.status === 'success').length
  const todayFailed = todayReleases.filter((r) => r.status === 'failed').length
  const todayRunning = todayReleases.filter((r) => r.status === 'running').length

  // 进行中的发布
  const runningReleases = releases.filter((r) => r.status === 'running')

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">总览</h1>
        <p className="text-muted-foreground text-sm">OpsHub 聚合维护平台运营概况</p>
      </div>

      {/* 第一行：统计卡片 */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">服务总数</CardTitle>
            <Server className="size-4 text-blue-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{serviceTotal}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">环境健康度</CardTitle>
            <Activity className="size-4 text-green-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {allEnvs.length > 0 ? `${healthyCount}/${allEnvs.length}` : '-'}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              {allEnvs.length > 0
                ? `${Math.round((healthyCount / allEnvs.length) * 100)}% 健康`
                : '加载中...'}
            </p>
          </CardContent>
        </Card>

        <Card
          className={`cursor-pointer hover:border-red-300 transition-colors ${alertStats && alertStats.total_open > 0 ? 'border-red-200/50' : ''}`}
          onClick={() => navigate('/alerts')}
        >
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">待处理告警</CardTitle>
            <Bell className={`size-4 ${alertStats && alertStats.total_open > 0 ? 'text-red-600' : 'text-muted-foreground'}`} />
          </CardHeader>
          <CardContent>
            <div className={`text-2xl font-bold ${alertStats && alertStats.total_open > 0 ? 'text-red-600' : ''}`}>
              {alertStats ? alertStats.total_open : '-'}
            </div>
            {alertStats && alertStats.p1_open > 0 && (
              <p className="text-xs text-red-600 mt-1">P1: {alertStats.p1_open} 条</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">异常环境</CardTitle>
            <AlertTriangle className="size-4 text-destructive" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{unhealthyEnvs.length}</div>
            {unknownCount > 0 && (
              <p className="text-xs text-muted-foreground mt-1">{unknownCount} 个状态未知</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">今日发布</CardTitle>
            <Rocket className="size-4 text-purple-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{todayReleases.length}</div>
            <p className="text-xs text-muted-foreground mt-1">
              {todaySuccess} 成功{todayFailed > 0 ? ` / ${todayFailed} 失败` : ''}{todayRunning > 0 ? ` / ${todayRunning} 进行中` : ''}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* 第二行：异常环境 + 最近发布 */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {/* 异常环境快速定位 */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base flex items-center gap-2">
              <AlertTriangle className="size-4 text-destructive" />
              异常环境
            </CardTitle>
          </CardHeader>
          <CardContent>
            {unhealthyEnvs.length === 0 ? (
              <div className="flex items-center gap-2 py-4 text-sm text-muted-foreground">
                <CheckCircle className="size-4 text-green-600" />
                所有环境运行正常
              </div>
            ) : (
              <div className="flex flex-col gap-2">
                {unhealthyEnvs.slice(0, 5).map((env) => (
                  <div
                    key={env.id}
                    className="flex items-center justify-between rounded-md border border-destructive/20 bg-destructive/5 p-3 cursor-pointer hover:bg-destructive/10 transition-colors"
                    onClick={() => navigate(`/services/${env.service_id}`)}
                  >
                    <div className="flex flex-col gap-0.5">
                      <span className="text-sm font-medium">
                        {env.service_name || env.service_id.slice(0, 8)} / {env.env_code}
                      </span>
                      <span className="text-xs text-muted-foreground">
                        {env.health_last_message || env.health_status}
                        {env.health_last_checked_at && ` · ${env.health_last_checked_at}`}
                      </span>
                    </div>
                    <Badge variant="destructive">
                      {env.health_status === 'unreachable' ? '不可达' : '异常'}
                    </Badge>
                  </div>
                ))}
                {unhealthyEnvs.length > 5 && (
                  <p className="text-xs text-muted-foreground text-center">
                    还有 {unhealthyEnvs.length - 5} 个异常环境...
                  </p>
                )}
              </div>
            )}
          </CardContent>
        </Card>

        {/* 最近发布动态 */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base flex items-center gap-2">
              <Rocket className="size-4 text-purple-600" />
              最近发布动态
            </CardTitle>
            <span
              className="text-xs text-muted-foreground cursor-pointer hover:text-foreground flex items-center gap-1"
              onClick={() => navigate('/releases')}
            >
              查看全部 <ArrowRight className="size-3" />
            </span>
          </CardHeader>
          <CardContent>
            {releases.length === 0 ? (
              <p className="text-sm text-muted-foreground py-4 text-center">暂无发布记录</p>
            ) : (
              <div className="flex flex-col gap-2">
                {releases.slice(0, 6).map((rel) => (
                  <div
                    key={rel.id}
                    className="flex items-center justify-between rounded-md border p-3 cursor-pointer hover:bg-muted/50 transition-colors"
                    onClick={() => navigate(`/releases/${rel.id}`)}
                  >
                    <div className="flex flex-col gap-0.5">
                      <span className="text-sm font-medium">
                        {rel.release_type === 'deploy' ? '🚀 部署' : '🔄 回滚'}{' '}
                        <span className="text-sm">{serviceNameMap.get(rel.service_id) || rel.service_id.slice(0, 8)}</span>
                      </span>
                      <span className="text-xs text-muted-foreground">
                        {rel.operator_id || '系统'} · {rel.created_at}
                      </span>
                    </div>
                    <StatusBadge status={rel.status} />
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* 第三行：进行中发布 */}
      {runningReleases.length > 0 && (
        <Card className="border-blue-200/50">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base flex items-center gap-2">
              <Loader2 className="size-4 text-blue-600 animate-spin" />
              进行中的发布
            </CardTitle>
            <span
              className="text-xs text-muted-foreground cursor-pointer hover:text-foreground flex items-center gap-1"
              onClick={() => navigate('/releases')}
            >
              查看全部 <ArrowRight className="size-3" />
            </span>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col gap-2">
              {runningReleases.slice(0, 5).map((rel) => (
                <div
                  key={rel.id}
                  className="flex items-center justify-between rounded-md border border-blue-200/30 bg-blue-50/50 dark:bg-blue-950/20 p-3 cursor-pointer hover:bg-blue-100/50 dark:hover:bg-blue-950/30 transition-colors"
                  onClick={() => navigate(`/releases/${rel.id}`)}
                >
                  <div className="flex flex-col gap-0.5">
                    <span className="text-sm font-medium">
                      {rel.release_type === 'deploy' ? '🚀 部署' : '🔄 回滚'}{' '}
                      {serviceNameMap.get(rel.service_id) || rel.service_id.slice(0, 8)}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      {rel.operator_id || '系统'} · 开始于 {rel.started_at ? new Date(rel.started_at).toLocaleTimeString() : rel.created_at}
                    </span>
                  </div>
                  <StatusBadge status={rel.status} />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { label: string; cls: string }> = {
    pending: { label: '待执行', cls: 'bg-yellow-100 text-yellow-800' },
    running: { label: '执行中', cls: 'bg-blue-100 text-blue-800' },
    success: { label: '成功', cls: 'bg-green-100 text-green-800' },
    failed: { label: '失败', cls: 'bg-red-100 text-red-800' },
    cancelled: { label: '已取消', cls: 'bg-gray-100 text-gray-800' },
  }
  const info = map[status] ?? { label: status, cls: 'bg-gray-100 text-gray-800' }
  return <span className={`rounded px-2 py-0.5 text-xs font-medium ${info.cls}`}>{info.label}</span>
}
