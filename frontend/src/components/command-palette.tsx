import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search, Server, Monitor, ArrowRight, FileText, Rocket, Key, Activity, Settings as SettingsIcon, Database } from 'lucide-react'
import { cn } from '@/lib/utils'
import { searchApi, type SearchHit } from '@/api/search'
// ─── 静态快捷动作（页面跳转） ────────────────────────────
interface QuickAction {
  id: string
  title: string
  subtitle: string
  url: string
  icon: React.ComponentType<{ className?: string }>
  keywords: string[]
}

const QUICK_ACTIONS: QuickAction[] = [
  { id: 'go-services', title: '服务管理', subtitle: '所有服务列表', url: '/services', icon: Server, keywords: ['service', '服务', '台账', '管理'] },
  { id: 'go-releases', title: '发布中心', subtitle: '查看 Jenkins 构建', url: '/releases', icon: Rocket, keywords: ['release', '发布', '构建', 'jenkins'] },
  { id: 'go-hosts', title: '主机管理', subtitle: 'SSH 主机列表', url: '/hosts', icon: Monitor, keywords: ['host', '主机', 'ssh'] },
  { id: 'go-monitor', title: '资源监控', subtitle: 'CPU / 内存 / 磁盘', url: '/monitor', icon: Activity, keywords: ['monitor', '监控', '资源'] },
  { id: 'go-backups', title: '数据备份', subtitle: '备份 / 迁移 / 对象同步', url: '/backups/tasks', icon: Database, keywords: ['backup', '备份', '迁移', '同步', 'minio', 'oss'] },
  { id: 'go-logs', title: '日志中心', subtitle: '检索日志', url: '/logs', icon: FileText, keywords: ['log', '日志'] },
  { id: 'go-credentials', title: '凭证管理', subtitle: 'SSH / API 密钥', url: '/credentials', icon: Key, keywords: ['credential', '凭证', '密钥'] },
  { id: 'go-settings', title: '系统设置', subtitle: '平台配置', url: '/settings', icon: SettingsIcon, keywords: ['setting', '设置'] },
]

const TYPE_META: Record<SearchHit['type'], { label: string; icon: React.ComponentType<{ className?: string }> }> = {
  service: { label: '服务', icon: Server },
  host:    { label: '主机', icon: Monitor },
}

// ─── 组件 ────────────────────────────────────────────────

interface CommandPaletteProps {
  open: boolean
  onClose: () => void
}

export function CommandPalette({ open, onClose }: CommandPaletteProps) {
  const navigate = useNavigate()
  const inputRef = useRef<HTMLInputElement>(null)
  const [query, setQuery] = useState('')
  const [hits, setHits] = useState<SearchHit[]>([])
  const [loading, setLoading] = useState(false)
  const [activeIndex, setActiveIndex] = useState(0)

  // 重置状态
  useEffect(() => {
    if (open) {
      setQuery('')
      setHits([])
      setActiveIndex(0)
      // 延迟聚焦，避免被关闭逻辑覆盖
      setTimeout(() => inputRef.current?.focus(), 30)
    }
  }, [open])

  // 远程搜索（300ms 防抖）
  useEffect(() => {
    if (!open) return
    const trimmed = query.trim()
    if (!trimmed) {
      setHits([])
      setLoading(false)
      return
    }
    setLoading(true)
    const timer = setTimeout(async () => {
      try {
        const res = await searchApi.search(trimmed)
        setHits(res.data?.items || [])
        setActiveIndex(0)
      } catch {
        setHits([])
      } finally {
        setLoading(false)
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [query, open])

  // 计算合并候选项：本地快捷动作 + 远程搜索结果
  const trimmed = query.trim().toLowerCase()
  const filteredQuickActions = trimmed
    ? QUICK_ACTIONS.filter(a =>
        a.title.toLowerCase().includes(trimmed) ||
        a.subtitle.toLowerCase().includes(trimmed) ||
        a.keywords.some(k => k.toLowerCase().includes(trimmed))
      )
    : QUICK_ACTIONS

  type Item =
    | { kind: 'quick'; data: QuickAction }
    | { kind: 'hit'; data: SearchHit }

  const items: Item[] = [
    ...filteredQuickActions.map(a => ({ kind: 'quick' as const, data: a })),
    ...hits.map(h => ({ kind: 'hit' as const, data: h })),
  ]

  const total = items.length

  // 键盘
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      e.preventDefault()
      onClose()
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActiveIndex(i => (total > 0 ? (i + 1) % total : 0))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActiveIndex(i => (total > 0 ? (i - 1 + total) % total : 0))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      const it = items[activeIndex]
      if (it) {
        navigate(it.data.url)
        onClose()
      }
    }
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-[60]">
      {/* 遮罩 */}
      <div
        className="fixed inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
        aria-hidden
      />
      {/* 面板 */}
      <div className="fixed inset-x-0 top-[15%] mx-auto w-full max-w-xl px-4">
        <div
          className="overflow-hidden rounded-xl border bg-background shadow-2xl"
          onKeyDown={onKeyDown}
        >
          {/* 输入框 */}
          <div className="flex items-center gap-2 border-b px-4 py-3">
            <Search className="size-4 text-muted-foreground" />
            <input
              ref={inputRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜索服务、主机…  (Esc 关闭)"
              className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
            />
            {loading && <span className="text-xs text-muted-foreground">搜索中…</span>}
          </div>

          {/* 结果列表 */}
          <div className="max-h-[60vh] overflow-y-auto py-2">
            {total === 0 ? (
              <div className="px-4 py-10 text-center text-sm text-muted-foreground">
                {trimmed ? '未匹配到结果' : '输入关键字开始搜索'}
              </div>
            ) : (
              <ItemList
                items={items}
                activeIndex={activeIndex}
                onHover={setActiveIndex}
                onSelect={(it) => {
                  navigate(it.data.url)
                  onClose()
                }}
              />
            )}
          </div>

          {/* 底部提示 */}
          <div className="flex items-center justify-between border-t px-4 py-2 text-xs text-muted-foreground">
            <span>↑↓ 选择 · Enter 跳转 · Esc 关闭</span>
            <span>Ctrl/Cmd + K</span>
          </div>
        </div>
      </div>
    </div>
  )
}

// ─── 子组件：分组渲染 ──────────────────────────────────────

function ItemList({
  items,
  activeIndex,
  onHover,
  onSelect,
}: {
  items: Array<{ kind: 'quick'; data: QuickAction } | { kind: 'hit'; data: SearchHit }>
  activeIndex: number
  onHover: (i: number) => void
  onSelect: (it: { kind: 'quick'; data: QuickAction } | { kind: 'hit'; data: SearchHit }) => void
}) {
  // 分组：先快捷动作，后搜索结果
  const quick = items.filter((it) => it.kind === 'quick')
  const hits = items.filter((it) => it.kind === 'hit')

  let runningIndex = 0
  return (
    <>
      {quick.length > 0 && (
        <Section title="快捷跳转">
          {quick.map((it) => {
            const idx = runningIndex++
            const a = it.data as QuickAction
            const Icon = a.icon
            return (
              <Row
                key={`q-${a.id}`}
                active={idx === activeIndex}
                onMouseEnter={() => onHover(idx)}
                onClick={() => onSelect(it)}
                icon={<Icon className="size-4 text-muted-foreground" />}
                title={a.title}
                subtitle={a.subtitle}
              />
            )
          })}
        </Section>
      )}
      {hits.length > 0 && (
        <Section title="搜索结果">
          {hits.map((it) => {
            const idx = runningIndex++
            const h = it.data as SearchHit
            const meta = TYPE_META[h.type]
            const Icon = meta.icon
            return (
              <Row
                key={`h-${h.type}-${h.id}`}
                active={idx === activeIndex}
                onMouseEnter={() => onHover(idx)}
                onClick={() => onSelect(it)}
                icon={<Icon className="size-4 text-muted-foreground" />}
                title={h.title}
                subtitle={`${meta.label} · ${h.subtitle}`}
              />
            )
          })}
        </Section>
      )}
    </>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="mb-2">
      <div className="px-4 py-1 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
        {title}
      </div>
      <div>{children}</div>
    </div>
  )
}

function Row({
  active, icon, title, subtitle, onClick, onMouseEnter,
}: {
  active: boolean
  icon: React.ReactNode
  title: string
  subtitle: string
  onClick: () => void
  onMouseEnter: () => void
}) {
  return (
    <button
      type="button"
      onMouseEnter={onMouseEnter}
      onClick={onClick}
      className={cn(
        'flex w-full items-center gap-3 px-4 py-2 text-left text-sm transition-colors',
        active ? 'bg-accent text-accent-foreground' : 'hover:bg-accent/50'
      )}
    >
      {icon}
      <span className="flex-1 truncate">
        <span className="block truncate">{title}</span>
        <span className="block truncate text-xs text-muted-foreground">{subtitle}</span>
      </span>
      <ArrowRight className="size-3 opacity-50" />
    </button>
  )
}

// ─── 全局快捷键 Hook ──────────────────────────────────────

export function useCommandPaletteShortcut(setOpen: (v: boolean) => void) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const isModK = (e.ctrlKey || e.metaKey) && (e.key === 'k' || e.key === 'K')
      if (isModK) {
        e.preventDefault()
        setOpen(true)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [setOpen])
}
