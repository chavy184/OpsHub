import { useEffect, useState, useCallback } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
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
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs'
import { http } from '@/lib/http'
import { toast } from 'sonner'
import {
  Edit,
  Plus,
  RefreshCw,
  Send,
  Trash2,
} from 'lucide-react'

// ==================== Types ====================

interface ChannelDTO {
  id: string
  name: string
  channel_type: 'email' | 'wecom_bot'
  config: string
  enabled: boolean
  created_at: string
  updated_at: string
}

interface RuleDTO {
  id: string
  event_type: string
  channel_id: string
  enabled: boolean
  filter: string
  created_at: string
  updated_at: string
}

interface LogDTO {
  id: string
  channel_id: string
  event_type: string
  title: string
  content: string
  status: 'sent' | 'failed'
  error_message: string
  created_at: string
}

// ==================== Constants ====================

const CHANNEL_TYPE_OPTIONS = [
  { value: 'email', label: '邮件' },
  { value: 'wecom_bot', label: '企微机器人' },
]

const EVENT_TYPE_OPTIONS = [
  { value: 'health_check_fail', label: '健康检查失败' },
  { value: 'health_check_ok', label: '健康检查恢复' },
  { value: 'deploy_success', label: '部署成功' },
  { value: 'deploy_fail', label: '部署失败' },
  { value: 'resource_alert', label: '资源告警' },
  { value: 'host_offline', label: '主机离线' },
]

const EVENT_TYPE_LABEL: Record<string, string> = Object.fromEntries(
  EVENT_TYPE_OPTIONS.map((o) => [o.value, o.label])
)

const CHANNEL_TYPE_LABEL: Record<string, string> = {
  email: '邮件',
  wecom_bot: '企微机器人',
}

// ==================== API ====================

const channelApi = {
  list: () => http.get<ChannelDTO[]>('/api/v1/notification/channels'),
  create: (data: Partial<ChannelDTO>) => http.post('/api/v1/notification/channels', data),
  update: (id: string, data: Partial<ChannelDTO>) => http.put(`/api/v1/notification/channels/${id}`, data),
  delete: (id: string) => http.del(`/api/v1/notification/channels/${id}`),
  test: (id: string) => http.post(`/api/v1/notification/channels/${id}/test`),
}

const ruleApi = {
  list: () => http.get<RuleDTO[]>('/api/v1/notification/rules'),
  create: (data: Partial<RuleDTO>) => http.post('/api/v1/notification/rules', data),
  update: (id: string, data: Partial<RuleDTO>) => http.put(`/api/v1/notification/rules/${id}`, data),
  delete: (id: string) => http.del(`/api/v1/notification/rules/${id}`),
}

const logApi = {
  list: (params: Record<string, string | number>) =>
    http.get<{ list: LogDTO[]; total: number; page: number; page_size: number }>(
      '/api/v1/notification/logs',
      params
    ),
}

// ==================== Page ====================

export default function NotificationsPage() {
  const [tab, setTab] = useState('channels')
  const [channels, setChannels] = useState<ChannelDTO[]>([])
  const [rules, setRules] = useState<RuleDTO[]>([])
  const [logs, setLogs] = useState<LogDTO[]>([])
  const [logTotal, setLogTotal] = useState(0)
  const [logPage, setLogPage] = useState(1)
  const [loading, setLoading] = useState(false)

  const [channelDialogOpen, setChannelDialogOpen] = useState(false)
  const [editingChannel, setEditingChannel] = useState<ChannelDTO | null>(null)

  const [ruleDialogOpen, setRuleDialogOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<RuleDTO | null>(null)
  const [deleteChannelConfirm, setDeleteChannelConfirm] = useState<ChannelDTO | null>(null)
  const [deleteRuleConfirm, setDeleteRuleConfirm] = useState<RuleDTO | null>(null)

  const channelMap = Object.fromEntries(channels.map((c) => [c.id, c.name]))

  // ---- Data Loading ----

  const loadChannels = useCallback(async () => {
    try {
      const res = await channelApi.list()
      setChannels(res.data || [])
    } catch {
      /* ignore */
    }
  }, [])

  const loadRules = useCallback(async () => {
    try {
      const res = await ruleApi.list()
      setRules(res.data || [])
    } catch {
      /* ignore */
    }
  }, [])

  const loadLogs = useCallback(async () => {
    setLoading(true)
    try {
      const res = await logApi.list({ page: logPage, page_size: 20 })
      setLogs(res.data?.list || [])
      setLogTotal(res.data?.total || 0)
    } catch {
      /* ignore */
    }
    setLoading(false)
  }, [logPage])

  useEffect(() => {
    loadChannels()
    loadRules()
  }, [loadChannels, loadRules])

  useEffect(() => {
    if (tab === 'logs') loadLogs()
  }, [tab, loadLogs])

  // ---- Channel Actions ----

  const handleSaveChannel = async (data: {
    name: string
    channel_type: 'email' | 'wecom_bot'
    config: string
    enabled: boolean
  }) => {
    try {
      if (editingChannel) {
        await channelApi.update(editingChannel.id, data)
        toast.success('渠道已更新')
      } else {
        await channelApi.create(data)
        toast.success('渠道已创建')
      }
      setChannelDialogOpen(false)
      setEditingChannel(null)
      loadChannels()
    } catch {
      toast.error('操作失败')
    }
  }

  const handleDeleteChannel = async (ch: ChannelDTO) => {
    setDeleteChannelConfirm(ch)
  }

  const confirmDeleteChannel = async () => {
    if (!deleteChannelConfirm) return
    try {
      await channelApi.delete(deleteChannelConfirm.id)
      toast.success('渠道已删除')
      loadChannels()
    } catch {
      toast.error('删除失败')
    } finally {
      setDeleteChannelConfirm(null)
    }
  }

  const handleTestChannel = async (id: string) => {
    try {
      await channelApi.test(id)
      toast.success('测试消息已发送')
    } catch {
      toast.error('测试发送失败')
    }
  }

  const handleToggleChannel = async (ch: ChannelDTO) => {
    try {
      await channelApi.update(ch.id, { enabled: !ch.enabled })
      loadChannels()
    } catch {
      toast.error('操作失败')
    }
  }

  // ---- Rule Actions ----

  const handleSaveRule = async (data: {
    event_type: string
    channel_id: string
    enabled: boolean
  }) => {
    try {
      if (editingRule) {
        await ruleApi.update(editingRule.id, data)
        toast.success('规则已更新')
      } else {
        await ruleApi.create(data)
        toast.success('规则已创建')
      }
      setRuleDialogOpen(false)
      setEditingRule(null)
      loadRules()
    } catch {
      toast.error('操作失败')
    }
  }

  const handleDeleteRule = async (rule: RuleDTO) => {
    setDeleteRuleConfirm(rule)
  }

  const confirmDeleteRule = async () => {
    if (!deleteRuleConfirm) return
    try {
      await ruleApi.delete(deleteRuleConfirm.id)
      toast.success('规则已删除')
      loadRules()
    } catch {
      toast.error('删除失败')
    } finally {
      setDeleteRuleConfirm(null)
    }
  }

  const handleToggleRule = async (rule: RuleDTO) => {
    try {
      await ruleApi.update(rule.id, { enabled: !rule.enabled })
      loadRules()
    } catch {
      toast.error('操作失败')
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">通知推送</h1>
          <p className="text-muted-foreground text-sm">管理通知渠道、规则和发送记录</p>
        </div>
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="channels">通知渠道</TabsTrigger>
          <TabsTrigger value="rules">通知规则</TabsTrigger>
          <TabsTrigger value="logs">通知记录</TabsTrigger>
        </TabsList>

        {/* ==================== 通知渠道 Tab ==================== */}
        <TabsContent value="channels">
          <div className="flex justify-end mb-4">
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={loadChannels}>
                <RefreshCw className="size-4 mr-1" /> 刷新
              </Button>
              <Button
                size="sm"
                onClick={() => {
                  setEditingChannel(null)
                  setChannelDialogOpen(true)
                }}
              >
                <Plus className="size-4 mr-1" /> 新建渠道
              </Button>
            </div>
          </div>

          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>渠道名称</TableHead>
                    <TableHead className="w-28">类型</TableHead>
                    <TableHead className="w-20">状态</TableHead>
                    <TableHead className="w-40">创建时间</TableHead>
                    <TableHead className="w-48">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {channels.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                        暂无通知渠道
                      </TableCell>
                    </TableRow>
                  ) : (
                    channels.map((ch) => (
                      <TableRow key={ch.id}>
                        <TableCell className="font-medium">{ch.name}</TableCell>
                        <TableCell>
                          <Badge variant="secondary">
                            {CHANNEL_TYPE_LABEL[ch.channel_type] || ch.channel_type}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <button
                            onClick={() => handleToggleChannel(ch)}
                            className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                              ch.enabled ? 'bg-green-600' : 'bg-muted-foreground/30'
                            }`}
                            aria-label={ch.enabled ? '已启用，点击禁用' : '已禁用，点击启用'}
                          >
                            <span
                              className={`inline-block size-3.5 rounded-full bg-white transition-transform ${
                                ch.enabled ? 'translate-x-4.5' : 'translate-x-1'
                              }`}
                            />
                          </button>
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground font-mono">
                          {ch.created_at}
                        </TableCell>
                        <TableCell>
                          <div className="flex gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => {
                                setEditingChannel(ch)
                                setChannelDialogOpen(true)
                              }}
                            >
                              <Edit className="size-3 mr-1" /> 编辑
                            </Button>
                            <Button variant="ghost" size="sm" onClick={() => handleTestChannel(ch.id)}>
                              <Send className="size-3 mr-1" /> 测试
                            </Button>
                            <Button variant="ghost" size="sm" onClick={() => handleDeleteChannel(ch)}>
                              <Trash2 className="size-3 mr-1" /> 删除
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          <Dialog open={channelDialogOpen} onOpenChange={setChannelDialogOpen}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>{editingChannel ? '编辑渠道' : '新建渠道'}</DialogTitle>
                <DialogDescription>配置通知推送渠道信息</DialogDescription>
              </DialogHeader>
              <ChannelForm
                initial={editingChannel}
                onSubmit={handleSaveChannel}
                onCancel={() => setChannelDialogOpen(false)}
              />
            </DialogContent>
          </Dialog>
        </TabsContent>

        {/* ==================== 通知规则 Tab ==================== */}
        <TabsContent value="rules">
          <div className="flex justify-end mb-4">
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={loadRules}>
                <RefreshCw className="size-4 mr-1" /> 刷新
              </Button>
              <Button
                size="sm"
                onClick={() => {
                  setEditingRule(null)
                  setRuleDialogOpen(true)
                }}
              >
                <Plus className="size-4 mr-1" /> 新建规则
              </Button>
            </div>
          </div>

          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>事件类型</TableHead>
                    <TableHead>通知渠道</TableHead>
                    <TableHead className="w-20">状态</TableHead>
                    <TableHead className="w-40">创建时间</TableHead>
                    <TableHead className="w-40">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rules.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                        暂无通知规则
                      </TableCell>
                    </TableRow>
                  ) : (
                    rules.map((rule) => (
                      <TableRow key={rule.id}>
                        <TableCell>
                          <Badge variant="outline">
                            {EVENT_TYPE_LABEL[rule.event_type] || rule.event_type}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-sm">
                          {channelMap[rule.channel_id] || rule.channel_id}
                        </TableCell>
                        <TableCell>
                          <button
                            onClick={() => handleToggleRule(rule)}
                            className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                              rule.enabled ? 'bg-green-600' : 'bg-muted-foreground/30'
                            }`}
                            aria-label={rule.enabled ? '已启用，点击禁用' : '已禁用，点击启用'}
                          >
                            <span
                              className={`inline-block size-3.5 rounded-full bg-white transition-transform ${
                                rule.enabled ? 'translate-x-4.5' : 'translate-x-1'
                              }`}
                            />
                          </button>
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground font-mono">
                          {rule.created_at}
                        </TableCell>
                        <TableCell>
                          <div className="flex gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => {
                                setEditingRule(rule)
                                setRuleDialogOpen(true)
                              }}
                            >
                              <Edit className="size-3 mr-1" /> 编辑
                            </Button>
                            <Button variant="ghost" size="sm" onClick={() => handleDeleteRule(rule)}>
                              <Trash2 className="size-3 mr-1" /> 删除
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          <Dialog open={ruleDialogOpen} onOpenChange={setRuleDialogOpen}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>{editingRule ? '编辑规则' : '新建规则'}</DialogTitle>
                <DialogDescription>配置通知触发规则</DialogDescription>
              </DialogHeader>
              <RuleForm
                initial={editingRule}
                channels={channels}
                onSubmit={handleSaveRule}
                onCancel={() => setRuleDialogOpen(false)}
              />
            </DialogContent>
          </Dialog>
        </TabsContent>

        {/* ==================== 通知记录 Tab ==================== */}
        <TabsContent value="logs">
          <div className="flex justify-end mb-4">
            <Button variant="outline" size="sm" onClick={loadLogs}>
              <RefreshCw className="size-4 mr-1" /> 刷新
            </Button>
          </div>

          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>事件类型</TableHead>
                    <TableHead>标题</TableHead>
                    <TableHead className="w-20">状态</TableHead>
                    <TableHead>通知渠道</TableHead>
                    <TableHead className="w-40">发送时间</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {loading ? (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                        加载中...
                      </TableCell>
                    </TableRow>
                  ) : logs.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                        暂无通知记录
                      </TableCell>
                    </TableRow>
                  ) : (
                    logs.map((log) => (
                      <TableRow key={log.id}>
                        <TableCell>
                          <Badge variant="outline">
                            {EVENT_TYPE_LABEL[log.event_type] || log.event_type}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <div className="font-medium text-sm">{log.title}</div>
                          {log.error_message && (
                            <div className="text-xs text-red-500 line-clamp-1 mt-0.5">
                              {log.error_message}
                            </div>
                          )}
                        </TableCell>
                        <TableCell>
                          <Badge variant={log.status === 'sent' ? 'default' : 'destructive'}>
                            {log.status === 'sent' ? '已发送' : '失败'}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-sm">
                          {channelMap[log.channel_id] || log.channel_id}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground font-mono">
                          {log.created_at}
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          {logTotal > 20 && (
            <div className="flex justify-center gap-2 mt-4">
              <Button variant="outline" size="sm" disabled={logPage <= 1} onClick={() => setLogPage((p) => p - 1)}>
                上一页
              </Button>
              <span className="flex items-center text-sm text-muted-foreground">
                第 {logPage} 页 / 共 {Math.ceil(logTotal / 20)} 页
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={logPage * 20 >= logTotal}
                onClick={() => setLogPage((p) => p + 1)}
              >
                下一页
              </Button>
            </div>
          )}
        </TabsContent>
      </Tabs>

      {/* 删除渠道确认弹窗 */}
      <Dialog open={!!deleteChannelConfirm} onOpenChange={(o) => { if (!o) setDeleteChannelConfirm(null) }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除渠道</DialogTitle>
            <DialogDescription>
              确定要删除渠道 <strong>{deleteChannelConfirm?.name}</strong> 吗？关联的通知规则可能会受影响，此操作不可撤销。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteChannelConfirm(null)}>取消</Button>
            <Button variant="destructive" onClick={confirmDeleteChannel}>删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 删除规则确认弹窗 */}
      <Dialog open={!!deleteRuleConfirm} onOpenChange={(o) => { if (!o) setDeleteRuleConfirm(null) }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除规则</DialogTitle>
            <DialogDescription>
              确定要删除事件类型为 <strong>{EVENT_TYPE_LABEL[deleteRuleConfirm?.event_type ?? ''] || deleteRuleConfirm?.event_type}</strong> 的通知规则吗？
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteRuleConfirm(null)}>取消</Button>
            <Button variant="destructive" onClick={confirmDeleteRule}>删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

// ==================== Channel Form ====================

interface EmailConfig {
  smtp_host: string
  smtp_port: string
  smtp_user: string
  smtp_password: string
  from_address: string
  to_addresses: string
}

interface WecomConfig {
  webhook_url: string
}

function parseChannelConfig(configStr: string, type: string): EmailConfig | WecomConfig {
  try {
    return JSON.parse(configStr)
  } catch {
    if (type === 'email') {
      return { smtp_host: '', smtp_port: '465', smtp_user: '', smtp_password: '', from_address: '', to_addresses: '' }
    }
    return { webhook_url: '' }
  }
}

function ChannelForm({
  initial,
  onSubmit,
  onCancel,
}: {
  initial: ChannelDTO | null
  onSubmit: (data: { name: string; channel_type: 'email' | 'wecom_bot'; config: string; enabled: boolean }) => void
  onCancel: () => void
}) {
  const [name, setName] = useState(initial?.name || '')
  const [channelType, setChannelType] = useState<'email' | 'wecom_bot'>(initial?.channel_type || 'email')
  const [enabled, setEnabled] = useState(initial?.enabled ?? true)

  const initialConfig = initial ? parseChannelConfig(initial.config, initial.channel_type) : null

  // Email fields
  const [smtpHost, setSmtpHost] = useState((initialConfig as EmailConfig)?.smtp_host || '')
  const [smtpPort, setSmtpPort] = useState((initialConfig as EmailConfig)?.smtp_port || '465')
  const [smtpUser, setSmtpUser] = useState((initialConfig as EmailConfig)?.smtp_user || '')
  const [smtpPassword, setSmtpPassword] = useState((initialConfig as EmailConfig)?.smtp_password || '')
  const [fromAddress, setFromAddress] = useState((initialConfig as EmailConfig)?.from_address || '')
  const [toAddresses, setToAddresses] = useState((initialConfig as EmailConfig)?.to_addresses || '')

  // Wecom fields
  const [webhookUrl, setWebhookUrl] = useState((initialConfig as WecomConfig)?.webhook_url || '')

  const buildConfigJson = (): string => {
    if (channelType === 'email') {
      return JSON.stringify({
        smtp_host: smtpHost,
        smtp_port: smtpPort,
        smtp_user: smtpUser,
        smtp_password: smtpPassword,
        from_address: fromAddress,
        to_addresses: toAddresses,
      })
    }
    return JSON.stringify({ webhook_url: webhookUrl })
  }

  const handleSubmit = () => {
    if (!name.trim()) {
      toast.error('请输入渠道名称')
      return
    }
    onSubmit({
      name: name.trim(),
      channel_type: channelType,
      config: buildConfigJson(),
      enabled,
    })
  }

  return (
    <div className="flex flex-col gap-4">
      <div>
        <Label>渠道名称</Label>
        <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="如：运维邮件组" />
      </div>
      <div>
        <Label>渠道类型</Label>
        <Select
          options={CHANNEL_TYPE_OPTIONS}
          value={channelType}
          onChange={(e) => setChannelType(e.target.value as 'email' | 'wecom_bot')}
          disabled={!!initial}
        />
      </div>

      {channelType === 'email' && (
        <>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <Label>SMTP 主机</Label>
              <Input value={smtpHost} onChange={(e) => setSmtpHost(e.target.value)} placeholder="smtp.example.com" />
            </div>
            <div>
              <Label>SMTP 端口</Label>
              <Input value={smtpPort} onChange={(e) => setSmtpPort(e.target.value)} placeholder="465" />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <Label>SMTP 用户</Label>
              <Input value={smtpUser} onChange={(e) => setSmtpUser(e.target.value)} placeholder="user@example.com" />
            </div>
            <div>
              <Label>SMTP 密码</Label>
              <Input
                type="password"
                value={smtpPassword}
                onChange={(e) => setSmtpPassword(e.target.value)}
                placeholder="••••••"
              />
            </div>
          </div>
          <div>
            <Label>发件地址</Label>
            <Input value={fromAddress} onChange={(e) => setFromAddress(e.target.value)} placeholder="noreply@example.com" />
          </div>
          <div>
            <Label>收件地址</Label>
            <Input
              value={toAddresses}
              onChange={(e) => setToAddresses(e.target.value)}
              placeholder="多个地址用逗号分隔"
            />
          </div>
        </>
      )}

      {channelType === 'wecom_bot' && (
        <div>
          <Label>Webhook URL</Label>
          <Input
            value={webhookUrl}
            onChange={(e) => setWebhookUrl(e.target.value)}
            placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=..."
          />
        </div>
      )}

      <div className="flex items-center gap-2">
        <Label>启用</Label>
        <button
          type="button"
          onClick={() => setEnabled(!enabled)}
          className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
            enabled ? 'bg-green-600' : 'bg-muted-foreground/30'
          }`}
          aria-label={enabled ? '已启用' : '已禁用'}
        >
          <span
            className={`inline-block size-3.5 rounded-full bg-white transition-transform ${
              enabled ? 'translate-x-4.5' : 'translate-x-1'
            }`}
          />
        </button>
      </div>

      <DialogFooter>
        <Button variant="outline" onClick={onCancel}>
          取消
        </Button>
        <Button onClick={handleSubmit}>{initial ? '保存' : '创建'}</Button>
      </DialogFooter>
    </div>
  )
}

// ==================== Rule Form ====================

function RuleForm({
  initial,
  channels,
  onSubmit,
  onCancel,
}: {
  initial: RuleDTO | null
  channels: ChannelDTO[]
  onSubmit: (data: { event_type: string; channel_id: string; enabled: boolean }) => void
  onCancel: () => void
}) {
  const [eventType, setEventType] = useState(initial?.event_type || EVENT_TYPE_OPTIONS[0]?.value || '')
  const [channelId, setChannelId] = useState(initial?.channel_id || '')
  const [enabled, setEnabled] = useState(initial?.enabled ?? true)

  const channelOptions = channels.map((c) => ({ value: c.id, label: c.name }))

  const handleSubmit = () => {
    if (!channelId) {
      toast.error('请选择通知渠道')
      return
    }
    onSubmit({ event_type: eventType, channel_id: channelId, enabled })
  }

  return (
    <div className="flex flex-col gap-4">
      <div>
        <Label>事件类型</Label>
        <Select
          options={EVENT_TYPE_OPTIONS}
          value={eventType}
          onChange={(e) => setEventType(e.target.value)}
        />
      </div>
      <div>
        <Label>通知渠道</Label>
        <Select
          options={channelOptions}
          value={channelId}
          onChange={(e) => setChannelId(e.target.value)}
          placeholder="选择渠道"
        />
      </div>
      <div className="flex items-center gap-2">
        <Label>启用</Label>
        <button
          type="button"
          onClick={() => setEnabled(!enabled)}
          className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
            enabled ? 'bg-green-600' : 'bg-muted-foreground/30'
          }`}
          aria-label={enabled ? '已启用' : '已禁用'}
        >
          <span
            className={`inline-block size-3.5 rounded-full bg-white transition-transform ${
              enabled ? 'translate-x-4.5' : 'translate-x-1'
            }`}
          />
        </button>
      </div>

      <DialogFooter>
        <Button variant="outline" onClick={onCancel}>
          取消
        </Button>
        <Button onClick={handleSubmit}>{initial ? '保存' : '创建'}</Button>
      </DialogFooter>
    </div>
  )
}
