import { useEffect, useState } from 'react'
import { useSettingStore } from '@/stores/setting-store'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { toast } from 'sonner'
import type { SystemSetting } from '@/types/api'

const CATEGORY_LABEL: Record<string, string> = {
  general: '通用',
  notification: '通知',
  auth: '认证',
  deploy: '部署',
  monitor: '资源监控',
}

export default function SettingsPage() {
  const settings = useSettingStore((s) => s.settings)
  const loading = useSettingStore((s) => s.loading)
  const fetchSettings = useSettingStore((s) => s.fetchSettings)
  const batchUpdate = useSettingStore((s) => s.batchUpdate)

  const [edits, setEdits] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    fetchSettings()
  }, [fetchSettings])

  // Group settings by category
  const grouped = settings.reduce<Record<string, SystemSetting[]>>((acc, s) => {
    const cat = s.category || 'general'
    if (!acc[cat]) acc[cat] = []
    acc[cat].push(s)
    return acc
  }, {})

  const handleChange = (key: string, value: string) => {
    setEdits((e) => ({ ...e, [key]: value }))
  }

  const handleSave = async () => {
    if (Object.keys(edits).length === 0) {
      toast.info('没有需要保存的修改')
      return
    }
    setSaving(true)
    try {
      const items = Object.entries(edits).map(([setting_key, value]) => {
        const orig = settings.find((s) => s.setting_key === setting_key)
        return {
          setting_key,
          value,
          value_type: orig?.value_type,
          category: orig?.category,
          description: orig?.description,
        }
      })
      await batchUpdate(items as Parameters<typeof batchUpdate>[0])
      setEdits({})
      toast.success('设置已保存')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const isDirty = Object.keys(edits).length > 0

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">系统设置</h1>
          <p className="text-muted-foreground text-sm">修改后点击保存生效</p>
        </div>
        <Button onClick={handleSave} disabled={!isDirty || saving}>
          {saving ? '保存中…' : '保存修改'}
        </Button>
      </div>

      {loading ? (
        <div className="flex flex-col gap-4">
          {[...Array(3)].map((_, i) => <Skeleton key={i} className="h-24 w-full rounded" />)}
        </div>
      ) : settings.length === 0 ? (
        <div className="flex flex-col items-center gap-3 py-16 text-muted-foreground">
          <p className="text-sm">暂无系统设置项</p>
          <p className="text-xs">后端可预置默认配置</p>
        </div>
      ) : (
        Object.entries(grouped).map(([category, items]) => (
          <div key={category} className="rounded-lg border">
            <div className="flex items-center gap-2 border-b px-4 py-3">
              <span className="font-semibold text-sm">{CATEGORY_LABEL[category] ?? category}</span>
              <Badge variant="outline" className="text-xs">{items.length} 项</Badge>
            </div>
            <div className="divide-y">
              {items.map((setting) => {
                const value = setting.setting_key in edits ? edits[setting.setting_key] : setting.value
                const changed = setting.setting_key in edits && edits[setting.setting_key] !== setting.value
                return (
                  <div key={setting.id} className="flex items-center gap-4 px-4 py-3">
                    <div className="w-56 shrink-0">
                      <div className="flex items-center gap-2">
                        <span className="font-mono text-xs">{setting.setting_key}</span>
                        {changed && <Badge variant="secondary" className="text-xs h-4">已修改</Badge>}
                      </div>
                      {setting.description && (
                        <p className="text-xs text-muted-foreground mt-0.5">{setting.description}</p>
                      )}
                    </div>
                    <div className="flex-1">
                      {setting.value_type === 'bool' ? (
                        <button
                          type="button"
                          onClick={() => handleChange(setting.setting_key, value === 'true' ? 'false' : 'true')}
                          className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                            value === 'true' ? 'bg-green-600' : 'bg-muted-foreground/30'
                          }`}
                          aria-label={value === 'true' ? '已启用' : '已禁用'}
                        >
                          <span
                            className={`inline-block size-4 rounded-full bg-white transition-transform ${
                              value === 'true' ? 'translate-x-5.5' : 'translate-x-1'
                            }`}
                          />
                        </button>
                      ) : setting.value_type === 'number' ? (
                        <Input
                          type="number"
                          value={value}
                          onChange={(e) => handleChange(setting.setting_key, e.target.value)}
                          className="font-mono text-sm"
                        />
                      ) : setting.value_type === 'json' ? (
                        <Textarea
                          value={value}
                          onChange={(e) => handleChange(setting.setting_key, e.target.value)}
                          className="font-mono text-sm min-h-[60px]"
                          rows={3}
                        />
                      ) : (
                        <Input
                          value={value}
                          onChange={(e) => handleChange(setting.setting_key, e.target.value)}
                          className="font-mono text-sm"
                        />
                      )}
                    </div>
                    <div className="w-16 text-xs text-muted-foreground text-right">
                      {setting.value_type}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        ))
      )}
    </div>
  )
}
