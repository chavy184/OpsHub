import { useCallback, useEffect, useMemo, useState } from 'react'
import { Copy, Image, Play, RefreshCw, Search } from 'lucide-react'
import { toast } from 'sonner'
import { HostsSubTabs } from '@/components/layout/hosts-sub-tabs'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { useHostStore } from '@/stores/host-store'
import { executeImageSync, listHostImages, listImageSyncRecords, type HostImage, type ImageSyncRecord } from '@/api/image-sync'

const MODE_OPTIONS = [
  { value: 'skip_if_exists', label: '目标存在则跳过' },
  { value: 'overwrite', label: '重新导入覆盖 tag' },
]

const STATUS_MAP: Record<string, { label: string; variant: 'default' | 'secondary' | 'outline' | 'destructive' }> = {
  running: { label: '执行中', variant: 'default' },
  success: { label: '成功', variant: 'outline' },
  skipped: { label: '已跳过', variant: 'secondary' },
  failed: { label: '失败', variant: 'destructive' },
}

function formatFileSize(bytes: number): string {
  if (!bytes) return '-'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

export default function ImageSyncPage() {
  const hosts = useHostStore((s) => s.hosts)
  const fetchHosts = useHostStore((s) => s.fetchHosts)
  const [records, setRecords] = useState<ImageSyncRecord[]>([])
  const [images, setImages] = useState<HostImage[]>([])
  const [imageQuery, setImageQuery] = useState('')
  const [loading, setLoading] = useState(true)
  const [imageLoading, setImageLoading] = useState(false)
  const [executing, setExecuting] = useState(false)
  const [form, setForm] = useState({
    source_host_id: '',
    target_host_id: '',
    image: '',
    mode: 'skip_if_exists',
  })

  useEffect(() => {
    fetchHosts()
  }, [fetchHosts])

  useEffect(() => {
    if (hosts.length === 0) return
    setForm((prev) => {
      if (prev.source_host_id || prev.target_host_id) return prev
      return {
        ...prev,
        source_host_id: hosts[0]?.id || '',
        target_host_id: hosts[1]?.id || hosts[0]?.id || '',
      }
    })
  }, [hosts])

  const fetchRecords = useCallback(async () => {
    setLoading(true)
    try {
      const data = await listImageSyncRecords({ page: 1, page_size: 50 })
      setRecords(data.list || [])
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchImages = useCallback(async () => {
    if (!form.source_host_id) {
      toast.error('请先选择源主机')
      return
    }
    setImageLoading(true)
    try {
      const data = await listHostImages(form.source_host_id)
      setImages(data || [])
      setImageQuery('')
      if ((data || []).length === 0) {
        toast.info('源主机暂无镜像')
      }
    } catch (err: any) {
      toast.error(err?.message || '获取镜像列表失败')
    } finally {
      setImageLoading(false)
    }
  }, [form.source_host_id])

  useEffect(() => {
    setImages([])
    setImageQuery('')
  }, [form.source_host_id])

  useEffect(() => {
    fetchRecords()
  }, [fetchRecords])

  const handleExecute = async () => {
    if (!form.source_host_id || !form.target_host_id || !form.image.trim()) {
      toast.error('请选择源主机、目标主机并填写镜像名称')
      return
    }
    if (form.source_host_id === form.target_host_id) {
      toast.error('源主机和目标主机不能相同')
      return
    }
    setExecuting(true)
    try {
      await executeImageSync({ ...form, image: form.image.trim() })
      toast.success('镜像同步已触发，后台执行中')
      setForm((prev) => ({ ...prev, image: '' }))
      fetchRecords()
    } catch (err: any) {
      toast.error(err?.message || '执行失败')
    } finally {
      setExecuting(false)
    }
  }

  const handleSelectImage = (image: HostImage) => {
    setForm((prev) => ({ ...prev, image: image.name }))
  }

  const filteredImages = useMemo(() => {
    const query = imageQuery.trim().toLowerCase()
    if (!query) return images
    const keywords = query.split(/\s+/).filter(Boolean)
    return images.filter((img) => {
      const haystack = [
        img.name,
        img.repository,
        img.tag,
        img.image_id,
        img.size,
        img.created_at,
      ].join(' ').toLowerCase()
      return keywords.every((keyword) => haystack.includes(keyword))
    })
  }, [images, imageQuery])

  return (
    <div className="space-y-6">
      <HostsSubTabs />

      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">镜像同步</h1>
          <p className="text-sm text-muted-foreground mt-1">将源主机已有 Docker 镜像同步到目标主机，手动执行，不自动触发</p>
        </div>
        <Button variant="outline" size="sm" onClick={fetchRecords}>
          <RefreshCw className="w-4 h-4 mr-1" /> 刷新记录
        </Button>
      </div>

      <div className="grid gap-4 rounded-md border p-4">
        <div className="grid grid-cols-4 gap-4">
          <div className="grid gap-2">
            <Label>源主机</Label>
            <Select
              value={form.source_host_id}
              options={hosts.map((h) => ({ value: h.id, label: `${h.name} (${h.host_address})` }))}
              onChange={(e) => setForm({ ...form, source_host_id: e.target.value })}
              placeholder="请选择"
            />
          </div>
          <div className="grid gap-2">
            <Label>目标主机</Label>
            <Select
              value={form.target_host_id}
              options={hosts.map((h) => ({ value: h.id, label: `${h.name} (${h.host_address})` }))}
              onChange={(e) => setForm({ ...form, target_host_id: e.target.value })}
              placeholder="请选择"
            />
          </div>
          <div className="grid gap-2">
            <Label>镜像名称</Label>
            <Input
              value={form.image}
              onChange={(e) => setForm({ ...form, image: e.target.value })}
              placeholder="nginx:1.25"
            />
          </div>
          <div className="grid gap-2">
            <Label>同步模式</Label>
            <Select value={form.mode} options={MODE_OPTIONS} onChange={(e) => setForm({ ...form, mode: e.target.value })} />
          </div>
        </div>
        <div className="flex justify-end">
          <Button onClick={handleExecute} disabled={executing || hosts.length < 2}>
            <Play className="w-4 h-4 mr-1" /> {executing ? '触发中...' : '执行同步'}
          </Button>
        </div>
      </div>

      <div className="grid gap-3 rounded-md border p-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-sm font-semibold">源主机镜像</h2>
            <p className="text-xs text-muted-foreground mt-1">先查看源主机已有镜像，再选择要同步的镜像</p>
          </div>
          <Button variant="outline" size="sm" onClick={fetchImages} disabled={imageLoading || !form.source_host_id}>
            <Image className="w-4 h-4 mr-1" /> {imageLoading ? '读取中...' : '查看镜像'}
          </Button>
        </div>
        <div className="relative max-w-sm">
          <Search className="pointer-events-none absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
          <Input
            value={imageQuery}
            onChange={(e) => setImageQuery(e.target.value)}
            className="pl-8"
            placeholder="搜索镜像名、tag、Image ID"
            disabled={images.length === 0}
          />
        </div>
        {imageLoading ? (
          <div className="space-y-2">
            {[1, 2, 3].map((i) => <Skeleton key={i} className="h-10 w-full" />)}
          </div>
        ) : images.length === 0 ? (
          <div className="text-center py-8 text-sm text-muted-foreground">请选择源主机后查看镜像</div>
        ) : filteredImages.length === 0 ? (
          <div className="text-center py-8 text-sm text-muted-foreground">没有匹配的镜像</div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>镜像</TableHead>
                <TableHead>Image ID</TableHead>
                <TableHead>大小</TableHead>
                <TableHead>创建时间</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredImages.map((img) => (
                <TableRow key={`${img.image_id}-${img.name}`}>
                  <TableCell className="font-mono text-xs">{img.name}</TableCell>
                  <TableCell className="font-mono text-xs">{img.image_id}</TableCell>
                  <TableCell className="text-xs">{img.size || '-'}</TableCell>
                  <TableCell className="text-xs">{img.created_at || '-'}</TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm" onClick={() => handleSelectImage(img)}>选择</Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>

      {loading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => <Skeleton key={i} className="h-12 w-full" />)}
        </div>
      ) : records.length === 0 ? (
        <div className="text-center py-16 text-muted-foreground">
          <Copy className="w-12 h-12 mx-auto mb-4 opacity-30" />
          <p>暂无镜像同步记录</p>
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>镜像</TableHead>
              <TableHead>源主机</TableHead>
              <TableHead>目标主机</TableHead>
              <TableHead>模式</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>大小</TableHead>
              <TableHead>耗时</TableHead>
              <TableHead>开始时间</TableHead>
              <TableHead>错误</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {records.map((record) => (
              <TableRow key={record.id}>
                <TableCell className="font-mono text-xs">{record.image}</TableCell>
                <TableCell className="text-xs">{record.source_host_name}</TableCell>
                <TableCell className="text-xs">{record.target_host_name}</TableCell>
                <TableCell className="text-xs">{MODE_OPTIONS.find((opt) => opt.value === record.mode)?.label || record.mode}</TableCell>
                <TableCell>
                  <Badge variant={STATUS_MAP[record.status]?.variant || 'outline'}>
                    {STATUS_MAP[record.status]?.label || record.status}
                  </Badge>
                </TableCell>
                <TableCell className="text-xs">{formatFileSize(record.image_size)}</TableCell>
                <TableCell className="text-xs">{record.duration ? `${record.duration}s` : '-'}</TableCell>
                <TableCell className="text-xs">{record.started_at}</TableCell>
                <TableCell className="text-xs text-destructive max-w-[240px] truncate" title={record.error}>{record.error || '-'}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  )
}
