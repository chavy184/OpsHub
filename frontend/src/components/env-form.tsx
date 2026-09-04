import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Select } from '@/components/ui/select'
import { Plus, Trash2, RefreshCw } from 'lucide-react'
import { useState, useEffect, useCallback } from 'react'
import { http } from '@/lib/http'

// ─── 共享常量 ──────────────────────────────────────────

export const ENV_CODE_OPTIONS = [
  { value: 'dev', label: '开发 (dev)' },
  { value: 'test', label: '测试 (test)' },
  { value: 'staging', label: '预发 (staging)' },
  { value: 'prod', label: '生产 (prod)' },
]

export const LOG_SOURCE_OPTIONS = [
  { value: '', label: '不配置' },
  { value: 'loki', label: 'loki — Loki HTTP API' },
  { value: 'docker', label: 'docker — Docker logs (SSH)' },
  { value: 'file', label: 'file — 文件路径 (SSH)' },
]

// ─── 日志源配置工具函数 ────────────────────────────────

export function buildLogSourceConfig(type_: string, fields: Record<string, string>): string {
  if (type_ === 'loki') {
    const endpoint = fields.endpoint ?? ''
    const label_key = fields.label_key ?? ''
    try {
      const vals = JSON.parse(fields.label_values_json || '[]')
      if (Array.isArray(vals) && vals.length > 0) {
        return JSON.stringify({ endpoint, label_key, label_values: vals })
      }
    } catch {}
    return JSON.stringify({ endpoint, label_key })
  }
  if (type_ === 'docker') {
    try {
      const list = JSON.parse(fields.containers_json || '[]')
      if (Array.isArray(list) && list.length > 0) {
        return JSON.stringify({ containers: list })
      }
    } catch {}
    if (fields.container) return JSON.stringify({ container: fields.container })
    return '{}'
  }
  if (type_ === 'file') {
    try {
      const list = JSON.parse(fields.files_json || '[]')
      if (Array.isArray(list) && list.length > 0) {
        return JSON.stringify({ files: list })
      }
    } catch {}
    if (fields.path) return JSON.stringify({ path: fields.path })
    return '{}'
  }
  return '{}'
}

export function parseLogSourceConfig(type_: string, config: string): Record<string, string> {
  try {
    const obj = JSON.parse(config)
    if (type_ === 'loki') {
      const endpoint = obj.endpoint ?? ''
      const label_key = obj.label_key ?? ''
      // 新格式: label_key + label_values
      if (Array.isArray(obj.label_values) && obj.label_values.length > 0) {
        return { endpoint, label_key, label_values_json: JSON.stringify(obj.label_values) }
      }
      // 旧格式: labels 对象 → 取第一个 key 作为 label_key，value 作为唯一 label_value
      if (obj.labels && typeof obj.labels === 'object') {
        const entries = Object.entries(obj.labels)
        if (entries.length > 0) {
          const [k, v] = entries[0]!
          return { endpoint, label_key: k, label_values_json: JSON.stringify([v]) }
        }
      }
      return { endpoint, label_key, label_values_json: '[]' }
    }
    if (type_ === 'docker') {
      if (Array.isArray(obj.containers) && obj.containers.length > 0) {
        return { containers_json: JSON.stringify(obj.containers) }
      }
      if (obj.container) {
        return { containers_json: JSON.stringify([{ name: 'default', container: obj.container }]) }
      }
      return { containers_json: '[]' }
    }
    if (type_ === 'file') {
      if (Array.isArray(obj.files) && obj.files.length > 0) {
        return { files_json: JSON.stringify(obj.files) }
      }
      if (obj.path) {
        return { files_json: JSON.stringify([{ name: 'default', path: obj.path }]) }
      }
      return { files_json: '[]' }
    }
  } catch {}
  return {}
}

// ─── 日志源配置编辑器 ─────────────────────────────────

export function LogSourceConfigEditor({ type_, fields, onChange }: {
  type_: string; fields: Record<string, string>; onChange: (f: Record<string, string>) => void
}) {
  if (type_ === 'loki') return (
    <LokiLabelSetsEditor fields={fields} onChange={onChange} />
  )
  if (type_ === 'docker') return (
    <DockerContainersEditor fields={fields} onChange={onChange} />
  )
  if (type_ === 'file') return (
    <FileSourcesEditor fields={fields} onChange={onChange} />
  )
  return null
}

// ─── Docker 多容器编辑器 ──────────────────────────────

interface DockerContainerItem {
  name: string
  container: string
}

function DockerContainersEditor({ fields, onChange }: {
  fields: Record<string, string>
  onChange: (f: Record<string, string>) => void
}) {
  const parseList = useCallback((): DockerContainerItem[] => {
    try {
      const arr = JSON.parse(fields.containers_json || '[]')
      return Array.isArray(arr) ? arr : []
    } catch {
      return []
    }
  }, [fields.containers_json])

  const [list, setList] = useState<DockerContainerItem[]>(parseList)

  useEffect(() => {
    setList(parseList())
  }, [parseList])

  const sync = (next: DockerContainerItem[]) => {
    setList(next)
    onChange({ ...fields, containers_json: JSON.stringify(next) })
  }

  const addRow = () => sync([...list, { name: '', container: '' }])

  const removeRow = (idx: number) => sync(list.filter((_, i) => i !== idx))

  const updateRow = (idx: number, key: keyof DockerContainerItem, val: string) => {
    const next = list.map((item, i) => i === idx ? { ...item, [key]: val } : item)
    sync(next)
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <Label>容器列表</Label>
        <Button type="button" variant="outline" size="sm" onClick={addRow}>
          <Plus className="size-3.5 mr-1" />
          添加容器
        </Button>
      </div>
      {list.length === 0 && (
        <p className="text-xs text-muted-foreground py-2">暂无容器，点击上方按钮添加</p>
      )}
      {list.map((item, idx) => (
        <div key={idx} className="flex items-center gap-2">
          <Input
            className="w-32"
            placeholder="显示名称"
            value={item.name}
            onChange={(e) => updateRow(idx, 'name', e.target.value)}
          />
          <Input
            className="flex-1"
            placeholder="容器名 / ID"
            value={item.container}
            onChange={(e) => updateRow(idx, 'container', e.target.value)}
          />
          <Button type="button" variant="ghost" size="icon" onClick={() => removeRow(idx)}>
            <Trash2 className="size-4 text-destructive" />
          </Button>
        </div>
      ))}
    </div>
  )
}

// ─── Loki 多标签编辑器 ──────────────────────────────

function LokiLabelSetsEditor({ fields, onChange }: {
  fields: Record<string, string>
  onChange: (f: Record<string, string>) => void
}) {
  const endpoint = fields.endpoint ?? ''
  const labelKey = fields.label_key ?? ''

  const parseValues = useCallback((): string[] => {
    try {
      const arr = JSON.parse(fields.label_values_json || '[]')
      return Array.isArray(arr) ? arr : []
    } catch {
      return []
    }
  }, [fields.label_values_json])

  const [values, setValues] = useState<string[]>(parseValues)
  const [availableLabels, setAvailableLabels] = useState<string[]>([])
  const [availableValues, setAvailableValues] = useState<string[]>([])
  const [loadingLabels, setLoadingLabels] = useState(false)
  const [loadingValues, setLoadingValues] = useState(false)

  useEffect(() => { setValues(parseValues()) }, [parseValues])

  const syncValues = (next: string[]) => {
    setValues(next)
    onChange({ ...fields, label_values_json: JSON.stringify(next) })
  }

  const setLabelKey = (key: string) => {
    onChange({ ...fields, label_key: key, label_values_json: '[]' })
    setValues([])
    setAvailableValues([])
    if (key) fetchLabelValues(key)
  }

  // 拉取可用 Label 名称
  const fetchLabels = useCallback(async () => {
    if (!endpoint) return
    setLoadingLabels(true)
    try {
      const res = await http.get<string[]>('/api/v1/logs/loki-labels', { endpoint })
      setAvailableLabels(res.data ?? [])
    } catch {
      setAvailableLabels([])
    } finally {
      setLoadingLabels(false)
    }
  }, [endpoint])

  // 拉取指定 Label 的可用值
  const fetchLabelValues = useCallback(async (label: string) => {
    if (!endpoint || !label) return
    setLoadingValues(true)
    try {
      const res = await http.get<string[]>('/api/v1/logs/loki-label-values', { endpoint, label })
      setAvailableValues(res.data ?? [])
    } catch {
      setAvailableValues([])
    } finally {
      setLoadingValues(false)
    }
  }, [endpoint])

  const addValue = (val: string) => {
    if (!val || values.includes(val)) return
    syncValues([...values, val])
  }

  const removeValue = (idx: number) => syncValues(values.filter((_, i) => i !== idx))

  return (
    <div className="flex flex-col gap-3">
      {/* Loki 地址 */}
      <div className="flex flex-col gap-1.5">
        <Label>Loki 地址</Label>
        <div className="flex items-center gap-2">
          <Input
            className="flex-1"
            placeholder="http://192.168.1.100:3100"
            value={endpoint}
            onChange={(e) => onChange({ ...fields, endpoint: e.target.value })}
          />
          <Button type="button" variant="outline" size="sm" onClick={fetchLabels} disabled={!endpoint || loadingLabels}>
            <RefreshCw className={`size-3.5 mr-1 ${loadingLabels ? 'animate-spin' : ''}`} />
            获取 Labels
          </Button>
        </div>
      </div>

      {/* Label Key 选择 */}
      <div className="flex flex-col gap-1.5">
        <Label>Label Key <span className="text-muted-foreground font-normal text-xs">（用于区分不同日志源的标签名）</span></Label>
        {availableLabels.length > 0 ? (
          <Select
            className="w-64"
            options={[{ value: '', label: '选择 Label...' }, ...availableLabels.map((l) => ({ value: l, label: l }))]}
            value={labelKey}
            onChange={(e) => setLabelKey(e.target.value)}
          />
        ) : (
          <Input
            className="w-64"
            placeholder="手动输入，如 app、container_name"
            value={labelKey}
            onChange={(e) => onChange({ ...fields, label_key: e.target.value })}
          />
        )}
      </div>

      {/* Label Values 列表 */}
      {labelKey && (
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <Label>Label Values <span className="text-muted-foreground font-normal text-xs">（每个值对应一个日志查看 Tab）</span></Label>
          </div>

          {values.length === 0 && (
            <p className="text-xs text-muted-foreground py-1">暂无值，从下方选择或手动添加</p>
          )}

          <div className="flex flex-wrap gap-2">
            {values.map((val, idx) => (
              <div key={idx} className="flex items-center gap-1 rounded-md border px-2 py-1 text-sm">
                <span>{val}</span>
                <Button type="button" variant="ghost" size="icon" className="size-5" onClick={() => removeValue(idx)}>
                  <Trash2 className="size-3 text-destructive" />
                </Button>
              </div>
            ))}
          </div>

          {/* 从自动获取的值中选择 */}
          {availableValues.length > 0 ? (
            <Select
              className="w-64"
              options={[{ value: '', label: '+ 添加值...' }, ...availableValues.filter((v) => !values.includes(v)).map((v) => ({ value: v, label: v }))]}
              value=""
              onChange={(e) => { if (e.target.value) addValue(e.target.value) }}
            />
          ) : (
            <div className="flex items-center gap-2">
              {loadingValues && <span className="text-xs text-muted-foreground">加载中...</span>}
              <Input
                className="w-48"
                placeholder="手动输入值"
                id="manual-loki-value"
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    const el = e.target as HTMLInputElement
                    if (el.value) { addValue(el.value); el.value = '' }
                  }
                }}
              />
              <Button type="button" variant="outline" size="sm" onClick={() => {
                const el = document.getElementById('manual-loki-value') as HTMLInputElement | null
                if (el?.value) { addValue(el.value); el.value = '' }
              }}>
                添加
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

// ─── File 多文件编辑器 ───────────────────────────────

interface FileSourceItem {
  name: string
  path: string
}

function FileSourcesEditor({ fields, onChange }: {
  fields: Record<string, string>
  onChange: (f: Record<string, string>) => void
}) {
  const parseList = useCallback((): FileSourceItem[] => {
    try {
      const arr = JSON.parse(fields.files_json || '[]')
      return Array.isArray(arr) ? arr : []
    } catch {
      return []
    }
  }, [fields.files_json])

  const [list, setList] = useState<FileSourceItem[]>(parseList)

  useEffect(() => { setList(parseList()) }, [parseList])

  const sync = (next: FileSourceItem[]) => {
    setList(next)
    onChange({ ...fields, files_json: JSON.stringify(next) })
  }

  const addRow = () => sync([...list, { name: '', path: '' }])
  const removeRow = (idx: number) => sync(list.filter((_, i) => i !== idx))

  const updateRow = (idx: number, key: keyof FileSourceItem, val: string) => {
    sync(list.map((item, i) => i === idx ? { ...item, [key]: val } : item))
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <Label>日志文件列表</Label>
        <Button type="button" variant="outline" size="sm" onClick={addRow}>
          <Plus className="size-3.5 mr-1" />
          添加文件
        </Button>
      </div>
      {list.length === 0 && (
        <p className="text-xs text-muted-foreground py-2">暂无文件，点击上方按钮添加</p>
      )}
      {list.map((item, idx) => (
        <div key={idx} className="flex items-center gap-2">
          <Input
            className="w-32"
            placeholder="显示名称"
            value={item.name}
            onChange={(e) => updateRow(idx, 'name', e.target.value)}
          />
          <Input
            className="flex-1"
            placeholder="/var/log/app.log"
            value={item.path}
            onChange={(e) => updateRow(idx, 'path', e.target.value)}
          />
          <Button type="button" variant="ghost" size="icon" onClick={() => removeRow(idx)}>
            <Trash2 className="size-4 text-destructive" />
          </Button>
        </div>
      ))}
    </div>
  )
}
