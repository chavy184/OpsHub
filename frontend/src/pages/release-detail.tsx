import { useEffect, useState } from 'react'
import { useParams, Link, useSearchParams } from 'react-router-dom'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { ArrowLeft, CheckCircle, XCircle, Clock, Loader2, SkipForward, Play, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import * as releasesApi from '@/api/releases'
import * as jenkinsApi from '@/api/jenkins'
import * as settingsApi from '@/api/settings'
import { useServiceStore } from '@/stores/service-store'
import type { ReleaseRecord, ReleaseStepLog, StepStatus } from '@/types/api'

const RELEASE_STATUS_MAP: Record<string, { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline' }> = {
  pending:   { label: '待执行', variant: 'outline' },
  running:   { label: '执行中', variant: 'secondary' },
  success:   { label: '成功',   variant: 'default' },
  failed:    { label: '失败',   variant: 'destructive' },
  cancelled: { label: '已取消', variant: 'outline' },
}

const STEP_ICON: Record<StepStatus, React.ReactNode> = {
  pending: <Clock className="size-4 text-muted-foreground" />,
  running: <Loader2 className="size-4 text-blue-500 animate-spin" />,
  success: <CheckCircle className="size-4 text-green-500" />,
  failed:  <XCircle className="size-4 text-destructive" />,
  skipped: <SkipForward className="size-4 text-muted-foreground" />,
}

const STEP_LABEL: Record<StepStatus, string> = {
  pending: '待执行', running: '执行中', success: '成功', failed: '失败', skipped: '跳过',
}

function formatDuration(ms: number) {
  if (!ms) return '-'
  return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(1)}s`
}

export default function ReleaseDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [searchParams] = useSearchParams()
  const fromService = searchParams.get('from_service')
  const backTo = fromService ? `/services/${fromService}` : '/releases'
  const backLabel = fromService ? '返回服务详情' : '返回发布列表'
  const [release, setRelease] = useState<ReleaseRecord | null>(null)
  const [steps, setSteps] = useState<ReleaseStepLog[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [confirmAction, setConfirmAction] = useState<'execute' | 'delete' | null>(null)
  const [acting, setActing] = useState(false)
  const [expandedStep, setExpandedStep] = useState<string | null>(null)
  const envs = useServiceStore((s) => s.envs)
  const fetchEnvs = useServiceStore((s) => s.fetchEnvs)

  const fetchData = async (showLoader = true) => {
    if (!id) return
    if (showLoader) setLoading(true)
    else setRefreshing(true)
    try {
      const [rec, stepList] = await Promise.all([
        releasesApi.getRelease(id),
        releasesApi.getReleaseSteps(id).catch(() => [] as ReleaseStepLog[]),
      ])
      setRelease(rec)
      setSteps(Array.isArray(stepList) ? stepList : [])
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  useEffect(() => { fetchData() }, [id])

  // 加载环境数据获取 jenkins_jobs
  useEffect(() => {
    if (release?.service_id) fetchEnvs(release.service_id)
  }, [release?.service_id, fetchEnvs])

  // 默认展开最后一个有输出的步骤
  useEffect(() => {
    if (steps.length > 0 && !expandedStep) {
      const lastWithOutput = [...steps].reverse().find((s) => s.output || s.error_output)
      if (lastWithOutput) setExpandedStep(lastWithOutput.id)
    }
  }, [steps])

  // 执行中自动 3s 刷新
  useEffect(() => {
    if (release?.status !== 'running') return
    const timer = setInterval(() => fetchData(false), 3000)
    return () => clearInterval(timer)
  }, [release?.status])

  const handleAction = async () => {
    if (!confirmAction || !id) return
    setActing(true)
    try {
      if (confirmAction === 'execute') {
        const rec = await releasesApi.executeRelease(id)
        setRelease(rec)
        toast.success(rec.status === 'success' ? '发布成功！' : '发布已开始执行')
        fetchData(false)
      } else {
        await releasesApi.deleteRelease(id)
        toast.success('发布记录已删除')
        window.location.href = backTo
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '操作失败')
    } finally {
      setActing(false)
      setConfirmAction(null)
    }
  }

  if (loading) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (!release) {
    return <div className="text-muted-foreground py-16 text-center">发布记录不存在</div>
  }

  const statusInfo = RELEASE_STATUS_MAP[release.status] ?? { label: release.status, variant: 'outline' as const }
  const canExecute = release.status === 'pending'
  const canDelete = release.status !== 'running'
  const totalDuration = steps.reduce((acc, s) => acc + (s.duration_ms || 0), 0)

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link to={backTo}>
          <Button variant="ghost" size="sm" className="gap-1">
            <ArrowLeft className="size-4" />
            <span className="text-xs">{backLabel}</span>
          </Button>
        </Link>
        <div className="flex-1">
          <div className="flex items-center gap-2 flex-wrap">
            <h1 className="text-xl font-bold tracking-tight">发布详情</h1>
            <Badge variant={statusInfo.variant}>{statusInfo.label}</Badge>
            {refreshing && <Loader2 className="size-4 animate-spin text-muted-foreground" />}
          </div>
          <p className="text-muted-foreground text-xs font-mono mt-0.5">{release.id}</p>
        </div>
        <div className="flex items-center gap-2">
          {canExecute && (
            <Button size="sm" onClick={() => setConfirmAction('execute')}>
              <Play className="size-4 mr-1" />
              执行发布
            </Button>
          )}
          {canDelete && (
            <Button size="sm" variant="destructive" onClick={() => setConfirmAction('delete')}>
              <Trash2 className="size-4 mr-1" />
              删除
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={() => fetchData(false)} disabled={refreshing}>
            刷新
          </Button>
        </div>
      </div>

      {/* 基本信息卡片 */}
      <div className="rounded-lg border divide-y">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-0 divide-x">
          <InfoCell label="发布类型" value={release.release_type === 'deploy' ? '部署' : '回滚'} />
          <InfoCell label="策略" value={release.strategy || '-'} />
          <InfoCell label="操作人" value={release.operator_id || '-'} />
          <InfoCell
            label="总耗时"
            value={totalDuration > 0 ? formatDuration(totalDuration) :
              (release.started_at && release.ended_at
                ? formatDuration(new Date(release.ended_at).getTime() - new Date(release.started_at).getTime())
                : '-')}
          />
        </div>
        <div className="grid grid-cols-2 gap-0 divide-x">
          <InfoCell label="开始时间" value={release.started_at ? new Date(release.started_at).toLocaleString() : '—'} />
          <InfoCell label="结束时间" value={release.ended_at ? new Date(release.ended_at).toLocaleString() : '—'} />
        </div>
        {release.error_message && (
          <div className="px-4 py-3">
            <span className="text-sm text-muted-foreground">错误信息：</span>
            <span className="text-sm text-destructive ml-1">{release.error_message}</span>
          </div>
        )}
      </div>

      {/* Jenkins 信息 */}
      {release.strategy === 'jenkins' && (
        <JenkinsInfoCard
          release={release}
          jenkinsJob={(() => {
            const env = envs.find((e) => e.id === release.env_id)
            try { const jobs = JSON.parse(env?.jenkins_jobs || '[]'); return jobs[0]?.job ?? '' } catch { return '' }
          })()}
        />
      )}

      {/* 步骤日志 */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-base font-semibold">执行步骤 {steps.length > 0 && `(${steps.length})`}</h2>
          {steps.length > 0 && (
            <span className="text-xs text-muted-foreground">
              成功 {steps.filter(s => s.step_status === 'success').length} /
              失败 {steps.filter(s => s.step_status === 'failed').length} /
              共 {steps.length}
            </span>
          )}
        </div>

        {steps.length === 0 ? (
          <div className="rounded-lg border p-10 text-center text-muted-foreground">
            {release.status === 'pending' ? (
              <>
                <Play className="mx-auto mb-2 size-8 opacity-30" />
                <p className="text-sm">发布尚未执行，点击右上角「执行发布」开始</p>
              </>
            ) : (
              <>
                <Clock className="mx-auto mb-2 size-8 opacity-30" />
                <p className="text-sm">暂无步骤日志</p>
              </>
            )}
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            {steps.map((step) => {
              const isExpanded = expandedStep === step.id
              const hasOutput = !!(step.output || step.error_output)
              return (
                <div key={step.id} className="rounded-lg border overflow-hidden">
                  <button
                    className="w-full flex items-center gap-3 px-4 py-3 bg-muted/30 hover:bg-muted/50 transition-colors text-left"
                    onClick={() => hasOutput && setExpandedStep(isExpanded ? null : step.id)}
                    disabled={!hasOutput}
                  >
                    {STEP_ICON[step.step_status] ?? STEP_ICON.pending}
                    <span className="font-medium text-sm flex-1">
                      <span className="text-muted-foreground mr-1.5">#{step.step_order}</span>
                      {step.step_name}
                    </span>
                    <span className="text-xs text-muted-foreground">{STEP_LABEL[step.step_status]}</span>
                    {step.started_at && (
                      <span className="text-xs text-muted-foreground hidden sm:block">
                        {new Date(step.started_at).toLocaleTimeString()}
                      </span>
                    )}
                    <span className="text-xs font-mono text-muted-foreground w-14 text-right">
                      {formatDuration(step.duration_ms)}
                    </span>
                    {hasOutput && (
                      <span className="text-xs text-muted-foreground ml-1">{isExpanded ? '▲' : '▼'}</span>
                    )}
                  </button>

                  {isExpanded && hasOutput && (
                    <div className="border-t bg-[#1e1e2e] px-4 py-3 font-mono text-xs max-h-80 overflow-y-auto" ref={isExpanded ? (el) => { if (el) el.scrollTop = el.scrollHeight } : undefined}>
                      {step.output && (
                        <pre className="whitespace-pre-wrap text-[#cdd6f4] leading-relaxed">{step.output}</pre>
                      )}
                      {step.error_output && (
                        <pre className="whitespace-pre-wrap text-[#f38ba8] mt-1 leading-relaxed">{step.error_output}</pre>
                      )}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* 操作确认弹窗 */}
      <Dialog open={!!confirmAction} onOpenChange={(o) => { if (!o) setConfirmAction(null) }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{confirmAction === 'execute' ? '确认执行发布' : '确认删除'}</DialogTitle>
            <DialogDescription>
              {confirmAction === 'execute'
                ? '将开始执行部署流程，请确保目标主机已就绪。执行过程中状态会自动刷新。'
                : '删除后该发布记录将不可恢复。'}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmAction(null)}>取消</Button>
            <Button
              variant={confirmAction === 'delete' ? 'destructive' : 'default'}
              onClick={handleAction}
              disabled={acting}
            >
              {acting ? '执行中…' : confirmAction === 'execute' ? '确认执行' : '确认删除'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function InfoCell({ label, value }: { label: string; value: string }) {
  return (
    <div className="px-4 py-3">
      <p className="text-xs text-muted-foreground mb-0.5">{label}</p>
      <p className="text-sm font-medium">{value}</p>
    </div>
  )
}

function JenkinsInfoCard({ release, jenkinsJob }: { release: ReleaseRecord; jenkinsJob: string }) {
  const [consoleOutput, setConsoleOutput] = useState<string | null>(null)
  const [consoleLoading, setConsoleLoading] = useState(false)
  const [showConsole, setShowConsole] = useState(false)
  const [jenkinsURL, setJenkinsURL] = useState('')

  useEffect(() => {
    settingsApi.getSettings('deploy').then((settings) => {
      const urlSetting = settings.find((s) => s.setting_key === 'jenkins.url')
      if (urlSetting?.value) setJenkinsURL(urlSetting.value.replace(/\/$/, ''))
    }).catch(() => {})
  }, [])

  const jenkinsParams: Record<string, string> = (() => {
    try { return release.jenkins_params ? JSON.parse(release.jenkins_params) : {} } catch { return {} }
  })()

  const paramEntries = Object.entries(jenkinsParams)

  // 构建 Jenkins 页面链接
  const buildLink = (() => {
    if (!jenkinsURL || !jenkinsJob || !release.jenkins_build_no) return ''
    const parts = jenkinsJob.split('/')
    const jobPath = '/view/' + parts[0] + '/job/' + parts.slice(1).join('/job/')
    return `${jenkinsURL}${jobPath}/${release.jenkins_build_no}`
  })()

  const loadConsole = async () => {
    if (!release.jenkins_build_no || !jenkinsJob) return
    setConsoleLoading(true)
    try {
      const output = await jenkinsApi.getConsoleOutput(jenkinsJob, release.jenkins_build_no)
      setConsoleOutput(output)
      setShowConsole(true)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '获取控制台输出失败')
    } finally {
      setConsoleLoading(false)
    }
  }

  return (
    <div className="rounded-lg border">
      <div className="px-4 py-3 border-b bg-muted/30">
        <h2 className="text-base font-semibold">Jenkins 构建信息</h2>
      </div>
      <div className="p-4 space-y-3">
        {release.jenkins_build_no > 0 && (
          <div className="flex items-center gap-2 text-sm">
            <span className="text-muted-foreground">构建编号:</span>
            {buildLink ? (
              <a href={buildLink} target="_blank" rel="noopener noreferrer" className="hover:underline">
                <Badge variant="secondary" className="font-mono">#{release.jenkins_build_no}</Badge>
              </a>
            ) : (
              <Badge variant="secondary" className="font-mono">#{release.jenkins_build_no}</Badge>
            )}
            {buildLink && (
              <a
                href={buildLink + '/console'}
                target="_blank"
                rel="noopener noreferrer"
                className="text-xs text-primary hover:underline"
              >
                Jenkins 控制台
              </a>
            )}
            <Button variant="ghost" size="sm" onClick={loadConsole} disabled={consoleLoading}>
              {consoleLoading ? <Loader2 className="size-3 animate-spin mr-1" /> : null}
              查看日志
            </Button>
          </div>
        )}

        {paramEntries.length > 0 && (
          <div>
            <p className="text-sm text-muted-foreground mb-2">构建参数:</p>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
              {paramEntries.map(([key, val]) => (
                <div key={key} className="rounded border px-3 py-2 text-sm bg-muted/20">
                  <span className="text-muted-foreground">{key}:</span>{' '}
                  <span className="font-mono">{val || '(空)'}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {showConsole && consoleOutput !== null && (
          <div>
            <div className="flex items-center justify-between mb-2">
              <p className="text-sm text-muted-foreground">控制台输出:</p>
              <Button variant="ghost" size="sm" onClick={() => setShowConsole(false)}>收起</Button>
            </div>
            <div className="bg-[#1e1e2e] rounded-lg px-4 py-3 font-mono text-xs max-h-96 overflow-y-auto">
              <pre className="whitespace-pre-wrap text-[#cdd6f4] leading-relaxed">{consoleOutput}</pre>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
