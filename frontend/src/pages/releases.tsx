import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useReleaseStore } from '@/stores/release-store'
import { useServiceStore } from '@/stores/service-store'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Select } from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Plus, Play, Trash2, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import * as jenkinsApi from '@/api/jenkins'
import { ApiError } from '@/lib/http'
import type { CreateReleasePayload, ReleaseStatus, JenkinsJobInfo, JenkinsParamDef } from '@/types/api'

const ERR_CODE_PROD_TARGET_BLOCKED = 3004

const STATUS_MAP: Record<string, { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline' }> = {
  pending: { label: '待执行', variant: 'outline' },
  running: { label: '执行中', variant: 'secondary' },
  success: { label: '成功', variant: 'default' },
  failed: { label: '失败', variant: 'destructive' },
  cancelled: { label: '已取消', variant: 'outline' },
}

const STATUS_OPTIONS = [
  { value: '', label: '全部状态' },
  { value: 'pending', label: '待执行' },
  { value: 'running', label: '执行中' },
  { value: 'success', label: '成功' },
  { value: 'failed', label: '失败' },
]

export default function ReleasesPage() {
  const releases = useReleaseStore((s) => s.releases)
  const total = useReleaseStore((s) => s.total)
  const loading = useReleaseStore((s) => s.loading)
  const query = useReleaseStore((s) => s.query)
  const fetchReleases = useReleaseStore((s) => s.fetchReleases)
  const setQuery = useReleaseStore((s) => s.setQuery)
  const createRelease = useReleaseStore((s) => s.createRelease)
  const executeRelease = useReleaseStore((s) => s.executeRelease)
  const deleteRelease = useReleaseStore((s) => s.deleteRelease)
  const services = useServiceStore((s) => s.services)
  const fetchServices = useServiceStore((s) => s.fetchServices)

  const [showCreate, setShowCreate] = useState(false)
  const [confirmAction, setConfirmAction] = useState<{ type: 'execute' | 'delete'; id: string } | null>(null)

  useEffect(() => {
    fetchReleases()
    fetchServices()
  }, [fetchReleases, fetchServices])

  const serviceMap = new Map(services.map((s) => [s.id, s.service_name]))

  const handleConfirmAction = async () => {
    if (!confirmAction) return
    try {
      if (confirmAction.type === 'execute') {
        const result = await executeRelease(confirmAction.id)
        toast.success(`发布${result.status === 'success' ? '成功' : '已开始执行'}`)
      } else {
        await deleteRelease(confirmAction.id)
        toast.success('发布记录已删除')
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '操作失败')
    } finally {
      setConfirmAction(null)
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">发布中心</h1>
          <p className="text-muted-foreground text-sm">共 {total} 条发布记录</p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus data-icon="inline-start" />
          新建发布
        </Button>
      </div>

      {/* 筛选 */}
      <div className="flex items-end gap-3">
        <Select
          options={[{ value: '', label: '全部服务' }, ...services.map((s) => ({ value: s.id, label: s.service_name }))]}
          value={query.service_id ?? ''}
          onChange={(e) => setQuery({ service_id: e.target.value || undefined, page: 1 })}
          className="w-40"
        />
        <Select
          options={STATUS_OPTIONS}
          value={query.status ?? ''}
          onChange={(e) => setQuery({ status: (e.target.value || undefined) as ReleaseStatus | undefined, page: 1 })}
          className="w-36"
        />
      </div>

      {/* 表格 */}
      {loading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : releases.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
          <p className="text-sm">暂无发布记录</p>
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>服务</TableHead>
              <TableHead>环境</TableHead>
              <TableHead>策略</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>开始时间</TableHead>
              <TableHead>错误信息</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {releases.map((rel) => {
              const statusInfo = STATUS_MAP[rel.status] ?? { label: rel.status, variant: 'outline' as const }
              return (
                <TableRow key={rel.id}>
                  <TableCell className="font-mono text-xs">
                    <Link to={`/releases/${rel.id}`} className="hover:underline text-primary">
                      {rel.id.slice(0, 8)}
                    </Link>
                  </TableCell>
                  <TableCell className="text-sm">
                    {serviceMap.get(rel.service_id) ?? '-'}
                  </TableCell>
                  <TableCell className="text-xs">{rel.env_code || '-'}</TableCell>
                  <TableCell className="text-xs">{rel.strategy}</TableCell>
                  <TableCell>
                    <Badge variant={statusInfo.variant}>{statusInfo.label}</Badge>
                  </TableCell>
                  <TableCell className="text-xs">{rel.started_at ?? '-'}</TableCell>
                  <TableCell className="max-w-48 truncate text-xs text-destructive">
                    {rel.error_message || '-'}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      {rel.status === 'pending' && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setConfirmAction({ type: 'execute', id: rel.id })}
                        >
                          <Play data-icon="inline-start" />
                          执行
                        </Button>
                      )}
                      {rel.status !== 'running' && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => setConfirmAction({ type: 'delete', id: rel.id })}
                        >
                          <Trash2 data-icon="inline-start" />
                          删除
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      )}

      {/* 分页 */}
      {total > query.page_size && (
        <div className="flex items-center justify-end gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={query.page <= 1}
            onClick={() => setQuery({ page: query.page - 1 })}
          >
            上一页
          </Button>
          <span className="text-sm text-muted-foreground">
            {query.page} / {Math.ceil(total / query.page_size)}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={query.page >= Math.ceil(total / query.page_size)}
            onClick={() => setQuery({ page: query.page + 1 })}
          >
            下一页
          </Button>
        </div>
      )}

      {/* 新建发布 */}
      <CreateReleaseDialog open={showCreate} onOpenChange={setShowCreate} onCreate={createRelease} />

      {/* 确认操作 */}
      <Dialog open={confirmAction !== null} onOpenChange={() => setConfirmAction(null)}>
        <DialogContent onClose={() => setConfirmAction(null)}>
          <DialogHeader>
            <DialogTitle>{confirmAction?.type === 'execute' ? '确认执行发布' : '确认删除'}</DialogTitle>
            <DialogDescription>
              {confirmAction?.type === 'execute'
                ? '确认后将开始执行部署流程，请确保目标环境就绪。'
                : '删除后该发布记录将不可恢复。'}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmAction(null)}>取消</Button>
            <Button
              variant={confirmAction?.type === 'delete' ? 'destructive' : 'default'}
              onClick={handleConfirmAction}
            >
              确认{confirmAction?.type === 'execute' ? '执行' : '删除'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

export function CreateReleaseDialog({
  open,
  onOpenChange,
  onCreate,
  defaultServiceId,
  lockService,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreate: (p: CreateReleasePayload) => Promise<unknown>
  defaultServiceId?: string
  lockService?: boolean
}) {
  const services = useServiceStore((s) => s.services)
  const fetchServices = useServiceStore((s) => s.fetchServices)
  const fetchEnvs = useServiceStore((s) => s.fetchEnvs)
  const envs = useServiceStore((s) => s.envs)

  const [serviceId, setServiceId] = useState(defaultServiceId ?? '')
  const [envId, setEnvId] = useState('')
  const [jenkinsParams, setJenkinsParams] = useState<Record<string, string>>({})
  const [jobInfo, setJobInfo] = useState<JenkinsJobInfo | null>(null)
  const [jobLoading, setJobLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [prodOverridePayload, setProdOverridePayload] = useState<CreateReleasePayload | null>(null)
  const [adminPassword, setAdminPassword] = useState('')

  // 打开时同步 defaultServiceId
  useEffect(() => {
    if (open && defaultServiceId) setServiceId(defaultServiceId)
  }, [open, defaultServiceId])

  // 从选中环境的 jenkins_jobs 判断是否为 Jenkins 模式
  const selectedEnv = envs.find((e) => e.id === envId)
  const jenkinsJobs: { name: string; job: string }[] = (() => {
    try { return selectedEnv?.jenkins_jobs ? JSON.parse(selectedEnv.jenkins_jobs) : [] } catch { return [] }
  })()
  const isJenkins = jenkinsJobs.length > 0

  useEffect(() => {
    if (open) fetchServices()
  }, [open, fetchServices])

  // 选择服务后加载环境
  useEffect(() => {
    if (!serviceId) return
    fetchEnvs(serviceId)
  }, [serviceId, fetchEnvs])

  // 选择环境后，如果是 Jenkins 模式，加载第一个 job 的信息
  useEffect(() => {
    if (!envId || !isJenkins) { setJobInfo(null); return }
    const jenkinsJob = jenkinsJobs[0]?.job
    if (!jenkinsJob) return
    setJobLoading(true)
    jenkinsApi
      .getJobInfo(jenkinsJob)
      .then((info) => {
        setJobInfo(info)
        const defaults: Record<string, string> = {}
        for (const p of info.parameters) defaults[p.name] = p.default_value ?? ''
        setJenkinsParams(defaults)
      })
      .catch(() => {
        setJobInfo(null)
      })
      .finally(() => setJobLoading(false))
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [envId, isJenkins])

  const resetForm = () => {
    setServiceId(defaultServiceId ?? '')
    setEnvId('')
    setJenkinsParams({})
    setJobInfo(null)
    setProdOverridePayload(null)
    setAdminPassword('')
  }

  const handleServiceChange = (id: string) => {
    setServiceId(id)
    setEnvId('')
    setJenkinsParams({})
    setJobInfo(null)
  }

  const handleEnvChange = (id: string) => {
    setEnvId(id)
    setJenkinsParams({})
    setJobInfo(null)
  }

  const handleSubmit = async () => {
    if (!serviceId) { toast.error('请选择服务'); return }
    if (!envId) { toast.error('请选择环境'); return }
    if (!isJenkins) { toast.error('当前仅支持 Jenkins 发布，请为环境配置 Jenkins Job'); return }
    const payload: CreateReleasePayload = { service_id: serviceId, env_id: envId, strategy: 'jenkins', jenkins_params: jenkinsParams }
    await submitPayload(payload)
  }

  const submitPayload = async (payload: CreateReleasePayload) => {
    setSubmitting(true)
    try {
      await onCreate(payload)
      toast.success('发布单创建成功')
      onOpenChange(false)
      resetForm()
    } catch (err) {
      const message = err instanceof Error ? err.message : '创建失败'
      if (!payload.force_prod_target && err instanceof ApiError && err.code === ERR_CODE_PROD_TARGET_BLOCKED) {
        setProdOverridePayload(payload)
      } else {
        toast.error(message)
      }
    } finally {
      setSubmitting(false)
    }
  }

  const handleForceSubmit = async () => {
    if (!prodOverridePayload) return
    if (!adminPassword.trim()) { toast.error('请输入 admin 密码'); return }
    await submitPayload({
      ...prodOverridePayload,
      force_prod_target: true,
      admin_password: adminPassword,
    })
  }

  const renderParamField = (param: JenkinsParamDef) => {
    const value = jenkinsParams[param.name] ?? ''
    const updateParam = (v: string) =>
      setJenkinsParams((prev) => ({ ...prev, [param.name]: v }))

    if (param.type === 'ChoiceParameterDefinition' && param.choices) {
      return (
        <Select
          options={param.choices.map((c) => ({ value: c, label: c }))}
          value={value}
          onChange={(e) => updateParam(e.target.value)}
        />
      )
    }
    if (param.type === 'BooleanParameterDefinition') {
      return (
        <Select
          options={[
            { value: 'true', label: 'true' },
            { value: 'false', label: 'false' },
          ]}
          value={value}
          onChange={(e) => updateParam(e.target.value)}
        />
      )
    }
    return (
      <Input
        value={value}
        onChange={(e) => updateParam(e.target.value)}
        placeholder={param.description || param.name}
      />
    )
  }

  return (
    <>
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)} className="max-w-md">
        <DialogHeader>
          <DialogTitle>新建发布</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-4 py-4">
          {/* 服务选择 */}
          <div className="flex flex-col gap-1.5">
            <Label>服务 *</Label>
            <Select
              options={services.map((s) => ({ value: s.id, label: `${s.service_name} (${s.service_key})` }))}
              placeholder="选择服务"
              value={serviceId}
              onChange={(e) => handleServiceChange(e.target.value)}
              disabled={lockService}
            />
          </div>

          {/* 环境选择（始终在服务选择后显示） */}
          {serviceId && (
            <div className="flex flex-col gap-1.5">
              <Label>环境 *</Label>
              <Select
                options={envs.map((e) => ({ value: e.id, label: e.env_code }))}
                placeholder="选择环境"
                value={envId}
                onChange={(e) => handleEnvChange(e.target.value)}
              />
            </div>
          )}

          {/* Jenkins 模式：环境配置了 jenkins_jobs */}
          {envId && isJenkins && (
            <>
              <div className="rounded-md border p-2 text-xs bg-muted/50">
                <span className="font-medium">Jenkins 发布模式</span> — {jenkinsJobs[0]?.name || jenkinsJobs[0]?.job}
              </div>
              {jobLoading ? (
                <div className="flex items-center gap-2 text-sm text-muted-foreground py-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  加载 Jenkins Job 信息...
                </div>
              ) : jobInfo ? (
                <>
                  {jobInfo.description && (
                    <p className="text-xs text-muted-foreground">{jobInfo.description}</p>
                  )}
                  {jobInfo.parameters.length > 0 && (
                    <div className="flex flex-col gap-3">
                      {jobInfo.parameters.map((param) => (
                        <div key={param.name} className="flex flex-col gap-1.5">
                          <Label>
                            {param.name}
                            {param.description && (
                              <span className="ml-1 text-xs text-muted-foreground font-normal">
                                ({param.description})
                              </span>
                            )}
                          </Label>
                          {renderParamField(param)}
                        </div>
                      ))}
                    </div>
                  )}
                </>
              ) : null}
            </>
          )}

          {/* 脚本发布已下线：环境未配置 jenkins_jobs 时提示 */}
          {envId && !isJenkins && (
            <div className="rounded-md border border-yellow-300 bg-yellow-50 p-3 text-xs text-yellow-800">
              该环境未配置 Jenkins Job，当前仅支持 Jenkins 发布。请在「服务详情 → 环境」中绑定 Jenkins Job 后重试。
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button onClick={handleSubmit} disabled={submitting || (isJenkins && jobLoading)}>
            {submitting ? '创建中...' : '创建发布单'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
    <Dialog open={!!prodOverridePayload} onOpenChange={(o) => { if (!o) { setProdOverridePayload(null); setAdminPassword('') } }}>
      <DialogContent onClose={() => { setProdOverridePayload(null); setAdminPassword('') }} className="max-w-md">
        <DialogHeader>
          <DialogTitle>强制使用 Prod 目标</DialogTitle>
          <DialogDescription>
            当前操作命中了数据库迁移/对象同步的线上目标保护。确认要继续时请输入 admin 密码。
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-1.5 py-2">
          <Label>admin 密码</Label>
          <Input
            type="password"
            value={adminPassword}
            onChange={(e) => setAdminPassword(e.target.value)}
            autoFocus
          />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => { setProdOverridePayload(null); setAdminPassword('') }}>取消</Button>
          <Button variant="destructive" onClick={handleForceSubmit} disabled={submitting}>
            {submitting ? '提交中...' : '确认强制创建'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
    </>
  )
}
