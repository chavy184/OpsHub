import { useEffect, useState, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useServiceStore } from '@/stores/service-store'
import { useHostStore } from '@/stores/host-store'
import { http, ApiError } from '@/lib/http'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Search, FileText, Terminal, Server } from 'lucide-react'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

// 日志源类型对应的配置说明
const LOG_SOURCE_HELP: Record<string, string> = {
  file: '通过 SSH 连接主机，执行 grep 搜索指定日志文件',
  loki: '调用 Loki HTTP API（query_range）进行日志查询',
  docker: '通过 SSH 连接主机，执行 docker logs 命令读取容器日志',
}

/** 解析 log_source_config JSON 返回可读摘要 */
function formatLogSourceConfig(type_: string, raw: string): string {
  try {
    const obj = JSON.parse(raw)
    if (type_ === 'loki') {
      const vals = parseLokiLabelValues(obj)
      if (vals.length > 0) {
        return `${obj.endpoint ?? ''} | ${obj.label_key ?? ''}=${vals.join(', ')}`
      }
      return obj.endpoint ?? ''
    }
    if (type_ === 'docker') {
      const containers = parseDockerContainers(obj)
      if (containers.length > 0) return `容器: ${containers.map((c) => c.name || c.container).join(', ')}`
      return '未配置容器'
    }
    if (type_ === 'file') {
      const files = parseFileSources(obj)
      if (files.length > 0) return `文件: ${files.map((f) => f.name || f.path).join(', ')}`
      return '未配置文件'
    }
  } catch {}
  return raw
}

/** 解析 docker 日志源配置中的容器列表（兼容新旧格式） */
function parseDockerContainers(obj: Record<string, unknown>): Array<{ name: string; container: string }> {
  if (Array.isArray(obj.containers) && obj.containers.length > 0) {
    return obj.containers as Array<{ name: string; container: string }>
  }
  if (typeof obj.container === 'string' && obj.container) {
    return [{ name: 'default', container: obj.container }]
  }
  return []
}

/** 解析 loki 日志源配置中的 label values（兼容新旧格式） */
function parseLokiLabelValues(obj: Record<string, unknown>): string[] {
  if (Array.isArray(obj.label_values) && obj.label_values.length > 0) {
    return obj.label_values as string[]
  }
  // 旧格式: labels 对象 → 取第一个 value
  if (obj.labels && typeof obj.labels === 'object') {
    return Object.values(obj.labels as Record<string, string>)
  }
  return []
}

/** 解析 file 日志源配置中的文件列表（兼容新旧格式） */
function parseFileSources(obj: Record<string, unknown>): Array<{ name: string; path: string }> {
  if (Array.isArray(obj.files) && obj.files.length > 0) {
    return obj.files as Array<{ name: string; path: string }>
  }
  if (typeof obj.path === 'string' && obj.path) {
    return [{ name: 'default', path: obj.path }]
  }
  return []
}

interface LogSearchForm {
  serviceId: string
  envId: string
  keyword: string
  level: string
  startTime: string
  endTime: string
}

const LEVEL_OPTIONS = [
  { value: '', label: '全部级别' },
  { value: 'ERROR', label: 'ERROR' },
  { value: 'WARN', label: 'WARN' },
  { value: 'INFO', label: 'INFO' },
  { value: 'DEBUG', label: 'DEBUG' },
]

export default function LogsPage() {
  const [searchParams] = useSearchParams()
  const services = useServiceStore((s) => s.services)
  const fetchServices = useServiceStore((s) => s.fetchServices)
  const envs = useServiceStore((s) => s.envs)
  const fetchEnvs = useServiceStore((s) => s.fetchEnvs)
  const hosts = useHostStore((s) => s.hosts)
  const fetchHosts = useHostStore((s) => s.fetchHosts)

  const [form, setForm] = useState<LogSearchForm>(() => ({
    serviceId: searchParams.get('serviceId') ?? '',
    envId: searchParams.get('envId') ?? '',
    keyword: '',
    level: '',
    startTime: '',
    endTime: '',
  }))
  const [searched, setSearched] = useState(false)
  const [searchToken, setSearchToken] = useState(0)
  const [limit, setLimit] = useState(100)

  // 初始加载服务列表
  useEffect(() => {
    fetchServices()
    fetchHosts()
  }, [fetchServices, fetchHosts])

  // 服务变化时拉取环境列表
  useEffect(() => {
    if (form.serviceId) {
      fetchEnvs(form.serviceId)
    }
  }, [form.serviceId, fetchEnvs])

  // 若 URL 带了 serviceId + envId，在 envs 加载完成后自动触发搜索
  useEffect(() => {
    const urlServiceId = searchParams.get('serviceId')
    const urlEnvId = searchParams.get('envId')
    if (!urlServiceId || !urlEnvId) return
    const matched = envs.find((e) => e.id === urlEnvId)
    if (!matched) return
    setSearched(true)
    setSearchToken((t) => (t === 0 ? 1 : t))
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [envs])

  const serviceOptions = [
    { value: '', label: '请选择服务' },
    ...services.map((s) => ({ value: s.id, label: `${s.service_name} (${s.service_key})` })),
  ]

  const envOptions = [
    { value: '', label: '请选择环境' },
    ...envs.map((e) => ({ value: e.id, label: `${e.env_code}${e.log_source_type ? ` [${e.log_source_type}]` : ''}` })),
  ]

  const selectedEnv = envs.find((e) => e.id === form.envId)
  const logSourceType = selectedEnv?.log_source_type
  const logSourceConfig = selectedEnv?.log_source_config
  // 仅 file/docker 类型会走 SSH，所以需要展示主机信息
  const sourceHost = useMemo(() => {
    if (!selectedEnv?.host_id) return null
    return hosts.find((h) => h.id === selectedEnv.host_id) ?? null
  }, [selectedEnv?.host_id, hosts])
  const needsHost = logSourceType === 'file' || logSourceType === 'docker'

  const handleSearch = () => {
    if (!form.serviceId || !form.envId) return
    setSearched(true)
    setSearchToken((token) => token + 1)
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">日志中心</h1>
        <p className="text-muted-foreground text-sm">按服务、环境搜索运行日志</p>
      </div>

      {/* 搜索条件 */}
      <div className="rounded-lg border p-4 flex flex-col gap-4">
        <div className="grid grid-cols-2 gap-4">
          <div className="flex flex-col gap-1.5">
            <Label>服务</Label>
            <Select
              options={serviceOptions}
              value={form.serviceId}
              onChange={(e) => setForm((f) => ({ ...f, serviceId: e.target.value, envId: '' }))}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>环境</Label>
            <Select
              options={form.serviceId ? envOptions : [{ value: '', label: '请先选择服务' }]}
              value={form.envId}
              onChange={(e) => setForm((f) => ({ ...f, envId: e.target.value }))}
              disabled={!form.serviceId}
            />
          </div>
        </div>

        {/* 日志源信息展示 */}
        {selectedEnv && (
          <div className="rounded-md bg-muted/50 p-3 text-sm flex items-start gap-3">
            <Terminal className="size-4 mt-0.5 shrink-0 text-muted-foreground" />
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-1 flex-wrap">
                <span className="text-muted-foreground">日志源：</span>
                {logSourceType ? (
                  <Badge variant="secondary">{logSourceType}</Badge>
                ) : (
                  <span className="text-muted-foreground italic">未配置日志源</span>
                )}
                {logSourceConfig && (
                  <span className="font-mono text-xs text-muted-foreground">
                    {formatLogSourceConfig(logSourceType ?? '', logSourceConfig)}
                  </span>
                )}
                {needsHost && (
                  sourceHost ? (
                    <span className="flex items-center gap-1 text-xs">
                      <Server className="size-3 text-muted-foreground" />
                      <Badge variant="outline" className="font-mono">
                        {sourceHost.name}
                      </Badge>
                      <span className="font-mono text-muted-foreground">{sourceHost.host_address}</span>
                      <Badge
                        variant={sourceHost.agent_status === 'online' ? 'default' : 'secondary'}
                        className="text-[10px] px-1.5 py-0"
                      >
                        {sourceHost.agent_status === 'online' ? '在线' : sourceHost.agent_status === 'offline' ? '离线' : '未知'}
                      </Badge>
                    </span>
                  ) : selectedEnv.host_id ? (
                    <span className="flex items-center gap-1 text-xs text-yellow-600">
                      <Server className="size-3" />主机不存在（host_id={selectedEnv.host_id.slice(0, 8)}…）
                    </span>
                  ) : (
                    <span className="flex items-center gap-1 text-xs text-yellow-600">
                      <Server className="size-3" />未绑定主机
                    </span>
                  )
                )}
              </div>
              {logSourceType && (
                <p className="text-xs text-muted-foreground">{LOG_SOURCE_HELP[logSourceType] ?? logSourceType}</p>
              )}
              {!logSourceType && (
                <p className="text-xs text-muted-foreground">
                  请在服务详情 → 环境 → 编辑 中配置日志源类型和地址，然后才能搜索日志。
                </p>
              )}
            </div>
          </div>
        )}

        <div className="grid grid-cols-3 gap-4">
          <div className="col-span-1 flex flex-col gap-1.5">
            <Label>日志级别</Label>
            <Select
              options={LEVEL_OPTIONS}
              value={form.level}
              onChange={(e) => setForm((f) => ({ ...f, level: e.target.value }))}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>开始时间</Label>
            <Input
              type="datetime-local"
              value={form.startTime}
              onChange={(e) => setForm((f) => ({ ...f, startTime: e.target.value }))}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>结束时间</Label>
            <Input
              type="datetime-local"
              value={form.endTime}
              onChange={(e) => setForm((f) => ({ ...f, endTime: e.target.value }))}
            />
          </div>
        </div>

        <div className="flex items-center gap-3">
          <div className="flex-1">
            <Input
              placeholder="关键词搜索（支持正则，如：ERROR|FATAL）"
              value={form.keyword}
              onChange={(e) => setForm((f) => ({ ...f, keyword: e.target.value }))}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            />
          </div>
          <Select
            options={[
              { value: '100', label: '100 条' },
              { value: '200', label: '200 条' },
              { value: '500', label: '500 条' },
              { value: '1000', label: '1000 条' },
            ]}
            value={String(limit)}
            onChange={(e) => setLimit(Number(e.target.value))}
            className="w-28"
          />
          <Button
            onClick={handleSearch}
            disabled={!form.serviceId || !form.envId || !logSourceType}
          >
            <Search className="size-4 mr-1" />
            搜索
          </Button>
        </div>
      </div>

      {/* 结果区域 */}
      {!searched ? (
        <div className="flex flex-col items-center gap-3 py-20 text-muted-foreground">
          <FileText className="size-12 opacity-20" />
          <p className="text-sm">选择服务和环境后点击搜索</p>
        </div>
      ) : !logSourceType ? (
        <div className="rounded-lg border border-dashed p-10 text-center">
          <Terminal className="mx-auto mb-3 size-10 opacity-20" />
          <p className="text-sm font-medium">该环境未配置日志源</p>
          <p className="text-xs text-muted-foreground mt-1">
            请前往服务详情 → 环境 Tab → 点击编辑，配置日志源类型（file / loki / docker）后再搜索
          </p>
        </div>
      ) : (
        <LogResultPanel form={form} env={selectedEnv!} searchToken={searchToken} limit={limit} />
      )}
    </div>
  )
}

function LogResultPanel({
  form,
  env,
  searchToken,
  limit,
}: {
  form: LogSearchForm
  env: NonNullable<ReturnType<typeof useServiceStore.getState>['envs'][0]>
  searchToken: number
  limit: number
}) {
  // Phase 1 的日志查询需要后端 SSH grep 支持，目前展示配置信息和操作指引
  const logSourceType = env.log_source_type
  const logSourceConfig = env.log_source_config

  // Docker 多容器支持
  const dockerContainers = useMemo(() => {
    if (logSourceType !== 'docker' || !logSourceConfig) return []
    try {
      return parseDockerContainers(JSON.parse(logSourceConfig))
    } catch {
      return []
    }
  }, [logSourceType, logSourceConfig])
  const [selectedContainer, setSelectedContainer] = useState('')

  // Loki 多标签值支持
  const lokiLabelValues = useMemo(() => {
    if (logSourceType !== 'loki' || !logSourceConfig) return []
    try {
      return parseLokiLabelValues(JSON.parse(logSourceConfig))
    } catch {
      return []
    }
  }, [logSourceType, logSourceConfig])
  const [selectedLabelSet, setSelectedLabelSet] = useState('')

  // File 多文件支持
  const fileSources = useMemo(() => {
    if (logSourceType !== 'file' || !logSourceConfig) return []
    try {
      return parseFileSources(JSON.parse(logSourceConfig))
    } catch {
      return []
    }
  }, [logSourceType, logSourceConfig])
  const [selectedFile, setSelectedFile] = useState('')

  // 源列表变化时默认选中第一个
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

  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lines, setLines] = useState<Array<{ts: number; line: string}>>([])

  useEffect(() => {
    if (searchToken === 0) return

    let mounted = true
    const fetchLogs = async () => {
      if (!form.serviceId || !form.envId) return
      if (!logSourceType) return
      if (logSourceType !== 'loki' && logSourceType !== 'docker' && logSourceType !== 'file') return
      setLoading(true)
      setError(null)
      setLines([])
      try {
        const body = {
          serviceId: form.serviceId,
          envId: form.envId,
          container: selectedContainer || undefined,
          file: selectedFile || undefined,
          labelSet: selectedLabelSet || undefined,
          keyword: form.keyword,
          level: form.level,
          startTime: form.startTime || undefined,
          endTime: form.endTime || undefined,
          limit,
        }
        const response = await http.post<any>('/api/v1/logs/search', body)
        if (!mounted) return

        const lokiPayload = response.data
        const lokiData = lokiPayload?.data
        const parsed: Array<{ts: number; line: string}> = []

        if (lokiData?.result && Array.isArray(lokiData.result)) {
          for (const stream of lokiData.result) {
            if (!stream?.values || !Array.isArray(stream.values)) {
              continue
            }

            for (const value of stream.values) {
              if (!Array.isArray(value) || value.length < 2) {
                continue
              }

              const rawTs = Number(value[0])
              const line = String(value[1])
              const ts = rawTs > 1e12 ? Math.floor(rawTs / 1e6) : rawTs
              parsed.push({ ts, line })
            }
          }
        }

        if (parsed.length === 0 && Array.isArray(lokiPayload?.result)) {
          for (const stream of lokiPayload.result) {
            if (!stream?.values || !Array.isArray(stream.values)) {
              continue
            }

            for (const value of stream.values) {
              if (!Array.isArray(value) || value.length < 2) {
                continue
              }

              const rawTs = Number(value[0])
              const line = String(value[1])
              const ts = rawTs > 1e12 ? Math.floor(rawTs / 1e6) : rawTs
              parsed.push({ ts, line })
            }
          }
        }

        parsed.sort((a, b) => b.ts - a.ts)
        if (mounted) setLines(parsed)
      } catch (e: unknown) {
        if (!mounted) return
        if (e instanceof ApiError) {
          setError(e.message)
          return
        }
        if (e instanceof Error) {
          setError(e.message)
          return
        }
        setError('日志查询失败')
      } finally {
        if (mounted) setLoading(false)
      }
    }

    fetchLogs()
    return () => { mounted = false }
  }, [searchToken, form.serviceId, form.envId, form.keyword, form.level, form.startTime, form.endTime, logSourceType, selectedContainer, selectedLabelSet, selectedFile])

  // 关键词高亮和统计
  const matchesCount = useMemo(() => {
    if (!form.keyword) return 0
    try {
      const esc = form.keyword.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
      const re = new RegExp(esc, 'gi')
      let cnt = 0
      for (const l of lines) {
        const m = String(l.line).match(re)
        if (m) cnt += m.length
      }
      return cnt
    } catch (_) {
      return 0
    }
  }, [lines, form.keyword])

  const renderHighlighted = (text: string, kw: string) => {
    if (!kw) return text
    try {
      const esc = kw.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
      const splitRe = new RegExp(`(${esc})`, 'gi')
      const tokenRe = new RegExp(`^${esc}$`, 'i')
      const parts = String(text).split(splitRe)
      return parts.map((p, i) => (
        tokenRe.test(p) ? <mark key={i} className="bg-yellow-200 text-black px-0.5">{p}</mark> : <span key={i}>{p}</span>
      ))
    } catch (_) {
      return text
    }
  }

  const grepCmd = logSourceType === 'file'
    ? `# SSH 连接主机后执行:\ngrep -rn "${form.keyword || 'ERROR'}" ${logSourceConfig || '/var/log/app/*.log'}${form.level ? ` | grep ${form.level}` : ''}`
    : null

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">日志查询</h3>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>日志源: {logSourceType}</span>
          <span>·</span>
          <span>{logSourceConfig}</span>
        </div>
      </div>

      {/* Docker 多容器 Tab 切换 */}
      {dockerContainers.length > 1 && (
        <Tabs value={selectedContainer} onValueChange={(v) => { setSelectedContainer(v) }}>
          <TabsList>
            {dockerContainers.map((ct) => (
              <TabsTrigger key={ct.container} value={ct.container}>
                {ct.name || ct.container}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      )}

      {/* Loki 多标签值 Tab 切换 */}
      {lokiLabelValues.length > 1 && (
        <Tabs value={selectedLabelSet} onValueChange={(v) => { setSelectedLabelSet(v) }}>
          <TabsList>
            {lokiLabelValues.map((val) => (
              <TabsTrigger key={val} value={val}>
                {val}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      )}

      {/* File 多文件 Tab 切换 */}
      {fileSources.length > 1 && (
        <Tabs value={selectedFile} onValueChange={(v) => { setSelectedFile(v) }}>
          <TabsList>
            {fileSources.map((f) => (
              <TabsTrigger key={f.path} value={f.path}>
                {f.name || f.path}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      )}

      {logSourceType === 'file' && grepCmd && (
        <div>
          <p className="text-xs text-muted-foreground mb-2">
            file 类型日志需通过 SSH 在主机上执行命令查询，以下是等效命令：
          </p>
          <div className="rounded-md bg-[#1e1e2e] p-4 font-mono text-xs">
            <pre className="text-[#cdd6f4] whitespace-pre-wrap">{grepCmd}</pre>
          </div>
          {form.startTime && (
            <p className="text-xs text-muted-foreground mt-2">
              时间范围：{form.startTime} ~ {form.endTime || '现在'}<br />
              提示：可用 <code className="bg-muted px-1 rounded">awk</code> 或 <code className="bg-muted px-1 rounded">sed</code> 按时间戳过滤日志行
            </p>
          )}
        </div>
      )}

      {logSourceType && (
        <div className="text-xs text-muted-foreground mb-2">
          默认展示最近 7 天的数据。可修改开始/结束时间以调整查询范围，关键词将高亮显示并统计匹配次数。
        </div>
      )}

      <div>
        <div className="mb-2 flex items-center justify-between">
          <div className="text-sm text-muted-foreground">查询结果</div>
          <div className="text-xs text-muted-foreground">{loading ? '加载中...' : `${lines.length} 行 • 匹配 ${matchesCount} 次`}</div>
        </div>

        {error && (
          <div className="rounded-md bg-red-50 text-red-700 p-3 text-sm mb-2">{error}</div>
        )}

        <div className="rounded border overflow-auto max-h-[420px] font-mono text-sm">
          {lines.length === 0 && !loading ? (
            <div className="p-6 text-center text-muted-foreground">无日志行</div>
          ) : (
            <ul>
              {lines.map((l, idx) => (
                <li key={idx} className="px-3 py-2 border-b last:border-b-0">
                  <div className="text-xs text-muted-foreground mb-1">{new Date(l.ts).toLocaleString()}</div>
                  <div className="whitespace-pre-wrap">{renderHighlighted(l.line, form.keyword)}</div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
