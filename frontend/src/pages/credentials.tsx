import { useEffect, useState } from 'react'
import { useCredentialStore } from '@/stores/credential-store'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select } from '@/components/ui/select'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Plus, Trash2, Key, Pencil } from 'lucide-react'
import { toast } from 'sonner'
import type { CredType, CreateCredentialPayload, UpdateCredentialPayload } from '@/types/api'

const CRED_TYPE_MAP: Record<CredType, { label: string; variant: 'default' | 'secondary' | 'outline' }> = {
  ssh_key: { label: 'SSH 密钥', variant: 'default' },
  ssh_password: { label: 'SSH 密码', variant: 'secondary' },
  token: { label: 'Token', variant: 'outline' },
}

const CRED_TYPE_OPTIONS = [
  { value: 'ssh_key', label: 'SSH 密钥' },
  { value: 'ssh_password', label: 'SSH 密码' },
  { value: 'token', label: 'Token/API Key' },
]

const SECRET_PLACEHOLDER: Record<CredType, string> = {
  ssh_key: '粘贴 SSH 私钥内容（请勿提交到仓库）',
  ssh_password: '密码',
  token: 'Token 值或 API Key',
}

export default function CredentialsPage() {
  const credentials = useCredentialStore((s) => s.credentials)
  const total = useCredentialStore((s) => s.total)
  const loading = useCredentialStore((s) => s.loading)
  const fetchCredentials = useCredentialStore((s) => s.fetchCredentials)
  const createCredential = useCredentialStore((s) => s.createCredential)
  const updateCredential = useCredentialStore((s) => s.updateCredential)
  const deleteCredential = useCredentialStore((s) => s.deleteCredential)

  const [showCreate, setShowCreate] = useState(false)
  const [editTarget, setEditTarget] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [form, setForm] = useState<CreateCredentialPayload>({
    name: '',
    cred_type: 'ssh_key',
    secret_data: '',
    passphrase: '',
    description: '',
  })
  const [editForm, setEditForm] = useState<UpdateCredentialPayload & { name: string }>({
    name: '',
    secret_data: '',
    passphrase: '',
    description: '',
  })

  useEffect(() => {
    fetchCredentials()
  }, [fetchCredentials])

  const resetForm = () =>
    setForm({ name: '', cred_type: 'ssh_key', secret_data: '', passphrase: '', description: '' })

  const handleCreate = async () => {
    if (!form.name.trim() || !form.secret_data.trim()) {
      toast.error('名称和凭证内容不能为空')
      return
    }
    setSubmitting(true)
    try {
      await createCredential(form)
      toast.success('凭证已创建')
      setShowCreate(false)
      resetForm()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteCredential(deleteTarget)
      toast.success('凭证已删除')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '删除失败')
    } finally {
      setDeleteTarget(null)
    }
  }

  const openEdit = (id: string) => {
    const cred = credentials.find((c) => c.id === id)
    if (!cred) return
    setEditForm({
      name: cred.name,
      secret_data: '',
      passphrase: '',
      description: cred.description || '',
    })
    setEditTarget(id)
  }

  const handleEdit = async () => {
    if (!editTarget || !editForm.name.trim()) {
      toast.error('名称不能为空')
      return
    }
    setSubmitting(true)
    try {
      const payload: UpdateCredentialPayload = {}
      if (editForm.name) payload.name = editForm.name
      if (editForm.secret_data) payload.secret_data = editForm.secret_data
      if (editForm.passphrase) payload.passphrase = editForm.passphrase
      if (editForm.description) payload.description = editForm.description
      await updateCredential(editTarget, payload)
      toast.success('凭证已更新')
      setEditTarget(null)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '更新失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">凭证管理</h1>
          <p className="text-muted-foreground text-sm">共 {total} 条凭证，凭证内容加密存储</p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus data-icon="inline-start" />
          新建凭证
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
                <TableHead>类型</TableHead>
                <TableHead>指纹</TableHead>
                <TableHead>描述</TableHead>
                <TableHead>创建时间</TableHead>
                <TableHead className="w-16">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {credentials.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center text-muted-foreground py-12">
                    <Key className="mx-auto mb-2 size-8 opacity-30" />
                    暂无凭证，点击右上角新建
                  </TableCell>
                </TableRow>
              ) : (
                credentials.map((c) => (
                  <TableRow key={c.id}>
                    <TableCell className="font-medium">{c.name}</TableCell>
                    <TableCell>
                      <Badge variant={CRED_TYPE_MAP[c.cred_type]?.variant ?? 'outline'}>
                        {CRED_TYPE_MAP[c.cred_type]?.label ?? c.cred_type}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs">{c.fingerprint ? c.fingerprint.slice(0, 24) + '…' : '-'}</TableCell>
                    <TableCell>{c.description || '-'}</TableCell>
                    <TableCell>{c.created_at ? new Date(c.created_at).toLocaleDateString() : '-'}</TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => openEdit(c.id)}
                        >
                          <Pencil className="size-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => setDeleteTarget(c.id)}
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      )}

      {/* 新建凭证 Dialog */}
      <Dialog open={showCreate} onOpenChange={(o) => { setShowCreate(o); if (!o) resetForm() }}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>新建凭证</DialogTitle>
            <DialogDescription>凭证内容将使用 AES-256-GCM 加密存储</DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-4 py-2">
            <div className="flex flex-col gap-1.5">
              <Label>名称 *</Label>
              <Input
                placeholder="例：prod-server-key"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>类型 *</Label>
              <Select
                value={form.cred_type}
                onChange={(e) => setForm((f) => ({ ...f, cred_type: e.target.value as CredType }))}
                options={CRED_TYPE_OPTIONS}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>凭证内容 *</Label>
              <textarea
                className="min-h-28 w-full rounded-md border bg-background px-3 py-2 text-sm font-mono resize-y focus:outline-none focus:ring-2 focus:ring-ring"
                placeholder={SECRET_PLACEHOLDER[form.cred_type]}
                value={form.secret_data}
                onChange={(e) => setForm((f) => ({ ...f, secret_data: e.target.value }))}
              />
            </div>
            {form.cred_type === 'ssh_key' && (
              <div className="flex flex-col gap-1.5">
                <Label>密码短语（Passphrase）</Label>
                <Input
                  type="password"
                  placeholder="如果私钥有密码保护，请填写密码短语"
                  value={form.passphrase ?? ''}
                  onChange={(e) => setForm((f) => ({ ...f, passphrase: e.target.value }))}
                />
              </div>
            )}
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
              {submitting ? '创建中…' : '创建'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 删除确认 */}
      <Dialog open={!!deleteTarget} onOpenChange={(o) => { if (!o) setDeleteTarget(null) }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>此凭证删除后不可恢复，已关联此凭证的主机将无法连接。</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>取消</Button>
            <Button variant="destructive" onClick={handleDelete}>确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 编辑凭证 Dialog */}
      <Dialog open={!!editTarget} onOpenChange={(o) => { if (!o) setEditTarget(null) }}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>编辑凭证</DialogTitle>
            <DialogDescription>修改凭证信息，凭证内容留空则不修改</DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-4 py-2">
            <div className="flex flex-col gap-1.5">
              <Label>名称 *</Label>
              <Input
                value={editForm.name}
                onChange={(e) => setEditForm((f) => ({ ...f, name: e.target.value }))}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>凭证内容（留空不修改）</Label>
              <textarea
                className="min-h-28 w-full rounded-md border bg-background px-3 py-2 text-sm font-mono resize-y focus:outline-none focus:ring-2 focus:ring-ring"
                placeholder="留空则保持原凭证内容不变"
                value={editForm.secret_data ?? ''}
                onChange={(e) => setEditForm((f) => ({ ...f, secret_data: e.target.value }))}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>密码短语（Passphrase，留空不修改）</Label>
              <Input
                type="password"
                placeholder="如果私钥有密码保护，请填写密码短语"
                value={editForm.passphrase ?? ''}
                onChange={(e) => setEditForm((f) => ({ ...f, passphrase: e.target.value }))}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>描述</Label>
              <Input
                value={editForm.description ?? ''}
                onChange={(e) => setEditForm((f) => ({ ...f, description: e.target.value }))}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditTarget(null)}>取消</Button>
            <Button onClick={handleEdit} disabled={submitting}>
              {submitting ? '保存中…' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
