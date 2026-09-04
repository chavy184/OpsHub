import { useEffect, useState, useCallback } from 'react'
import { useHostStore } from '@/stores/host-store'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { HostsSubTabs } from '@/components/layout/hosts-sub-tabs'
import {
  RefreshCw, Play, Square, RotateCcw, Settings, Plus, Trash2, FileText, Save,
} from 'lucide-react'
import { toast } from 'sonner'
import * as containerApi from '@/api/containers'
import type { Container, ContainerStatus } from '@/types/api'

const STATUS_MAP: Record<ContainerStatus, { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline' }> = {
  running: { label: '运行中', variant: 'default' },
  exited: { label: '已停止', variant: 'destructive' },
  paused: { label: '已暂停', variant: 'secondary' },
  restarting: { label: '重启中', variant: 'outline' },
  removed: { label: '已移除', variant: 'outline' },
  unknown: { label: '未知', variant: 'outline' },
}

export default function ContainersPage() {
  const hosts = useHostStore((s) => s.hosts)
  const fetchHosts = useHostStore((s) => s.fetchHosts)

  const [selectedHostId, setSelectedHostId] = useState<string>('')
  const [containers, setContainers] = useState<Container[]>([])
  const [loading, setLoading] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [actionLoading, setActionLoading] = useState<string | null>(null)

  // 配置管理对话框
  const [configContainer, setConfigContainer] = useState<Container | null>(null)
  const [showConfigDialog, setShowConfigDialog] = useState(false)
  const [configPaths, setConfigPaths] = useState<string[]>([])
  const [newPath, setNewPath] = useState('')
  const [description, setDescription] = useState('')
  const [pathChecking, setPathChecking] = useState(false)

  // 配置文件编辑对话框
  const [editingConfig, setEditingConfig] = useState<{ container: Container; path: string } | null>(null)
  const [configContent, setConfigContent] = useState('')
  const [originalContent, setOriginalContent] = useState('')
  const [configLoading, setConfigLoading] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    fetchHosts()
  }, [fetchHosts])

  useEffect(() => {
    if (hosts.length > 0 && !selectedHostId) {
      setSelectedHostId(hosts[0]!.id)
    }
  }, [hosts, selectedHostId])

  const fetchContainers = useCallback(async () => {
    if (!selectedHostId) return
    setLoading(true)
    try {
      const data = await containerApi.listContainers(selectedHostId)
      setContainers(data || [])
    } catch {
      setContainers([])
    } finally {
      setLoading(false)
    }
  }, [selectedHostId])

  useEffect(() => {
    fetchContainers()
  }, [fetchContainers])

  const handleSync = async () => {
    if (!selectedHostId) return
    setSyncing(true)
    try {
      const data = await containerApi.syncContainers(selectedHostId)
      setContainers(data ?? [])
      toast.success('容器列表同步成功')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '同步失败')
    } finally {
      setSyncing(false)
    }
  }

  const handleAction = async (id: string, action: 'start' | 'stop' | 'restart') => {
    setActionLoading(id)
    try {
      if (action === 'start') await containerApi.startContainer(selectedHostId, id)
      else if (action === 'stop') await containerApi.stopContainer(selectedHostId, id)
      else await containerApi.restartContainer(selectedHostId, id)
      toast.success(`容器${action === 'start' ? '启动' : action === 'stop' ? '停止' : '重启'}成功`)
      await fetchContainers()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '操作失败')
    } finally {
      setActionLoading(null)
    }
  }

  // 配置路径管理
  const openConfigManager = (c: Container) => {
    setConfigContainer(c)
    setConfigPaths([...(c.config_paths || [])])
    setDescription(c.description || '')
    setShowConfigDialog(true)
  }

  const handleAddPath = async () => {
    const path = newPath.trim()
    if (!path) return
    if (!path.startsWith('/')) {
      toast.error('必须是绝对路径（以 / 开头）')
      return
    }
    if (path.includes('..')) {
      toast.error('路径不允许包含 ..')
      return
    }
    if (configPaths.includes(path)) {
      toast.error('路径已存在')
      return
    }

    // 检查路径是否存在（仅容器运行中时验证）
    if (configContainer) {
      if (configContainer.status === 'running') {
        setPathChecking(true)
        try {
          await containerApi.readConfig(selectedHostId, configContainer.id, path)
          setConfigPaths((prev) => [...prev, path])
          setNewPath('')
          toast.success('路径验证通过，已添加')
        } catch (err) {
          toast.error(err instanceof Error ? err.message : '该配置文件在容器中不存在，请检查路径')
        } finally {
          setPathChecking(false)
        }
      } else {
        // 容器未运行时无法验证，直接添加
        setConfigPaths((prev) => [...prev, path])
        setNewPath('')
        toast.info('容器未运行，路径已添加（未验证是否存在）')
      }
    }
  }

  const handleRemovePath = (idx: number) => {
    setConfigPaths(configPaths.filter((_, i) => i !== idx))
  }

  const handleSaveConfig = async () => {
    if (!configContainer) {
      toast.error('未选择容器')
      return
    }
    // 如果输入框有未添加的路径，自动合并
    let pathsToSave = [...configPaths]
    const pending = newPath.trim()
    if (pending && pending.startsWith('/') && !pending.includes('..') && !pathsToSave.includes(pending)) {
      pathsToSave = [...pathsToSave, pending]
    }
    try {
      await containerApi.updateContainer(selectedHostId, configContainer.id, {
        config_paths: pathsToSave,
        description,
      })
      toast.success('容器配置已保存')
      setShowConfigDialog(false)
      setNewPath('')
      await fetchContainers()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    }
  }

  // 配置文件编辑
  const openConfigEditor = async (c: Container, path: string) => {
    setEditingConfig({ container: c, path })
    setConfigLoading(true)
    try {
      const data = await containerApi.readConfig(selectedHostId, c.id, path)
      setConfigContent(data.content)
      setOriginalContent(data.content)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '读取配置失败')
      setEditingConfig(null)
    } finally {
      setConfigLoading(false)
    }
  }

  const handleSaveConfig2 = async (restart: boolean) => {
    if (!editingConfig) return
    if (configContent === originalContent) {
      toast.info('内容未修改')
      return
    }
    setSaving(true)
    try {
      await containerApi.writeConfig(selectedHostId, editingConfig.container.id, {
        path: editingConfig.path,
        content: configContent,
        restart,
      })
      toast.success(restart ? '配置已保存，容器已重启' : '配置已保存')
      setEditingConfig(null)
      await fetchContainers()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  // 过滤
  const filtered = statusFilter
    ? containers.filter((c) => c.status === statusFilter)
    : containers

  return (
    <div className="space-y-4">
      <HostsSubTabs />

      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">容器管理</h2>
      </div>

      {/* 主机选择 & 过滤 */}
      <div className="flex items-center gap-3 flex-wrap">
        <div className="flex items-center gap-2">
          <Label className="text-sm whitespace-nowrap">宿主机</Label>
          <Select
            value={selectedHostId}
            onChange={(e) => setSelectedHostId(e.target.value)}
            className="w-56"
            options={hosts.map((h) => ({ value: h.id, label: `${h.name} (${h.host_address})` }))}
          />
        </div>

        <div className="flex items-center gap-2">
          <Label className="text-sm whitespace-nowrap">状态</Label>
          <Select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="w-32"
            options={[
              { value: '', label: '全部' },
              { value: 'running', label: '运行中' },
              { value: 'exited', label: '已停止' },
              { value: 'paused', label: '已暂停' },
              { value: 'removed', label: '已移除' },
            ]}
          />
        </div>

        <Button size="sm" variant="outline" onClick={handleSync} disabled={syncing || !selectedHostId}>
          <RefreshCw className={`size-4 mr-1 ${syncing ? 'animate-spin' : ''}`} />
          {syncing ? '同步中...' : '同步容器'}
        </Button>
      </div>

      {/* 容器列表 */}
      {loading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-12 w-full" />)}
        </div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-12 text-muted-foreground">
          {containers.length === 0
            ? selectedHostId ? '暂无容器数据，点击「同步容器」获取' : '请先选择宿主机'
            : '没有符合筛选条件的容器'}
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>容器名称</TableHead>
              <TableHead>镜像</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>配置文件</TableHead>
              <TableHead>最后同步</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filtered.map((c) => {
              const st = STATUS_MAP[c.status] || STATUS_MAP.unknown
              return (
                <TableRow key={c.id}>
                  <TableCell>
                    <div>
                      <div className="font-medium">{c.container_name}</div>
                      {c.description && (
                        <div className="text-xs text-muted-foreground">{c.description}</div>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground max-w-[200px] truncate">
                    {c.image || '-'}
                  </TableCell>
                  <TableCell>
                    <Badge variant={st.variant}>{st.label}</Badge>
                  </TableCell>
                  <TableCell>
                    {(c.config_paths || []).length > 0 ? (
                      <div className="flex flex-col gap-0.5">
                        {(c.config_paths || []).map((p) => (
                          <button
                            key={p}
                            className="text-xs text-primary hover:underline text-left truncate max-w-[180px]"
                            onClick={() => openConfigEditor(c, p)}
                            title={p}
                          >
                            <FileText className="size-3 inline mr-1" />{p.split('/').pop()}
                          </button>
                        ))}
                      </div>
                    ) : (
                      <span className="text-xs text-muted-foreground">未配置</span>
                    )}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {c.last_synced_at || '-'}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      {c.status !== 'running' && c.status !== 'removed' && (
                        <Button
                          size="sm" variant="ghost"
                          disabled={actionLoading === c.id}
                          onClick={() => handleAction(c.id, 'start')}
                          title="启动"
                        >
                          <Play className="size-3.5" />
                        </Button>
                      )}
                      {c.status === 'running' && (
                        <>
                          <Button
                            size="sm" variant="ghost"
                            disabled={actionLoading === c.id}
                            onClick={() => handleAction(c.id, 'stop')}
                            title="停止"
                          >
                            <Square className="size-3.5" />
                          </Button>
                          <Button
                            size="sm" variant="ghost"
                            disabled={actionLoading === c.id}
                            onClick={() => handleAction(c.id, 'restart')}
                            title="重启"
                          >
                            <RotateCcw className="size-3.5" />
                          </Button>
                        </>
                      )}
                      <Button
                        size="sm" variant="ghost"
                        onClick={() => openConfigManager(c)}
                        title="配置管理"
                      >
                        <Settings className="size-3.5" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      )}

      {/* 配置路径管理对话框 */}
      <Dialog open={showConfigDialog} onOpenChange={setShowConfigDialog}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>配置管理 - {configContainer?.container_name}</DialogTitle>
            <DialogDescription>管理容器内的配置文件路径和备注信息</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div>
              <Label>容器备注</Label>
              <Input
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="填写容器用途说明"
              />
            </div>

            <div>
              <Label>配置文件路径</Label>
              <div className="space-y-2 mt-1">
                {configPaths.map((p, idx) => (
                  <div key={p} className="flex items-center gap-2">
                    <Input value={p} readOnly className="text-sm flex-1" />
                    <Button
                      size="sm" variant="ghost"
                      onClick={() => handleRemovePath(idx)}
                    >
                      <Trash2 className="size-3.5 text-destructive" />
                    </Button>
                  </div>
                ))}
              </div>

              <div className="flex items-center gap-2 mt-2">
                <Input
                  value={newPath}
                  onChange={(e) => setNewPath(e.target.value)}
                  placeholder="/etc/nginx/nginx.conf"
                  className="text-sm flex-1"
                  onKeyDown={(e) => e.key === 'Enter' && handleAddPath()}
                />
                <Button size="sm" onClick={handleAddPath} disabled={pathChecking}>
                  <Plus className="size-3.5 mr-1" />
                  {pathChecking ? '验证中...' : '添加'}
                </Button>
              </div>
              <p className="text-xs text-muted-foreground mt-1">
                添加时会验证路径在容器中是否真实存在
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowConfigDialog(false)}>取消</Button>
            <Button onClick={handleSaveConfig}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 配置文件编辑对话框 */}
      <Dialog open={!!editingConfig} onOpenChange={(open) => !open && setEditingConfig(null)}>
        <DialogContent className="max-w-4xl max-h-[85vh] flex flex-col">
          <DialogHeader>
            <DialogTitle>
              编辑配置 - {editingConfig?.container.container_name}
            </DialogTitle>
            <DialogDescription className="truncate">
              {editingConfig?.path}
            </DialogDescription>
          </DialogHeader>

          {configLoading ? (
            <div className="flex-1 flex items-center justify-center py-12">
              <RefreshCw className="size-5 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <div className="flex-1 min-h-0">
              <Textarea
                value={configContent}
                onChange={(e) => setConfigContent(e.target.value)}
                className="w-full h-[50vh] font-mono text-sm resize-none"
                spellCheck={false}
              />
              {configContent !== originalContent && (
                <p className="text-xs text-amber-600 mt-1">
                  ⚠ 配置已修改，请选择仅保存或保存并重启容器。
                </p>
              )}
            </div>
          )}

          <DialogFooter>
            <Button variant="outline" onClick={() => setEditingConfig(null)}>取消</Button>
            <Button
              variant="outline"
              onClick={() => handleSaveConfig2(false)}
              disabled={saving || configContent === originalContent}
            >
              <Save className="size-4 mr-1" />
              {saving ? '保存中...' : '仅保存'}
            </Button>
            <Button
              onClick={() => handleSaveConfig2(true)}
              disabled={saving || configContent === originalContent}
            >
              <RotateCcw className="size-4 mr-1" />
              {saving ? '保存中...' : '保存并重启'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
