import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { cn } from '@/lib/utils'
import {
  Server,
  Settings,
  Key,
  Monitor,
  ScrollText,
  Rocket,
  ChevronLeft,
  ChevronRight,
  Activity,
  Send,
  Search,
  Database,
  FileText,
  Archive,
  LogOut,
} from 'lucide-react'
import { useState } from 'react'
import { Toaster } from 'sonner'
import { CommandPalette, useCommandPaletteShortcut } from '@/components/command-palette'
import { clearAuth, getUsername } from '@/lib/auth'

type NavItem = {
  to: string
  icon: React.ComponentType<{ className?: string }>
  label: string
  activePrefix?: string
}

const NAV_GROUPS: Array<{ label: string; items: NavItem[] }> = [
  {
    label: '应用',
    items: [
      { to: '/services', icon: Server, label: '服务管理' },
      { to: '/releases', icon: Rocket, label: '发布中心' },
      { to: '/logs', icon: ScrollText, label: '日志中心' },
    ],
  },
  {
    label: '基础设施',
    items: [
      { to: '/hosts', icon: Monitor, label: '主机管理', activePrefix: '/hosts' },
      { to: '/monitor', icon: Activity, label: '资源监控' },
      { to: '/backups/tasks', icon: Database, label: '数据备份', activePrefix: '/backups' },
      { to: '/documents', icon: FileText, label: '文档管理' },
      { to: '/archives', icon: Archive, label: '解压目录' },
    ],
  },
  {
    label: '系统',
    items: [
      { to: '/notifications', icon: Send, label: '通知推送' },
      { to: '/credentials', icon: Key, label: '凭证管理' },
      { to: '/settings', icon: Settings, label: '系统设置' },
    ],
  },
]

export default function AppLayout() {
  const [collapsed, setCollapsed] = useState(false)
  const [paletteOpen, setPaletteOpen] = useState(false)
  const location = useLocation()
  useCommandPaletteShortcut(setPaletteOpen)

  const handleLogout = () => {
    clearAuth()
    window.location.href = '/login'
  }

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      {/* Sidebar */}
      <aside
        className={cn(
          'flex flex-col border-r bg-sidebar text-sidebar-foreground transition-all duration-200',
          collapsed ? 'w-16' : 'w-56'
        )}
      >
        {/* Logo */}
        <div className="flex h-14 items-center gap-2 border-b px-4">
          <div className="flex size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground font-bold text-sm">
            O
          </div>
          {!collapsed && <span className="font-semibold text-sm">OpsHub</span>}
        </div>

        {/* Nav */}
        <nav className="flex-1 overflow-y-auto p-2">
          <div className="flex flex-col gap-3">
            {NAV_GROUPS.map((group, gi) => (
              <div key={group.label} className="flex flex-col gap-1">
                {!collapsed && (
                  <div className="px-3 pt-1 pb-0.5 text-[10px] font-semibold uppercase tracking-wider text-sidebar-foreground/40">
                    {group.label}
                  </div>
                )}
                {collapsed && gi > 0 && (
                  <div className="mx-3 my-1 border-t border-sidebar-border/40" />
                )}
                <ul className="flex flex-col gap-0.5">
                  {group.items.map(({ to, icon: Icon, label, activePrefix }) => (
                    <li key={to}>
                      <NavLink
                        to={to}
                        title={collapsed ? label : undefined}
                        className={({ isActive }) =>
                          cn(
                            'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                            isActive || (activePrefix && location.pathname.startsWith(activePrefix))
                              ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                              : 'text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground'
                          )
                        }
                      >
                        <Icon className="size-4 shrink-0" />
                        {!collapsed && <span className="truncate">{label}</span>}
                      </NavLink>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </nav>

        {/* Collapse toggle */}
        <button
          onClick={() => setCollapsed((c) => !c)}
          className="flex h-10 items-center justify-center border-t text-muted-foreground hover:text-foreground transition-colors"
          aria-label={collapsed ? '展开侧边栏' : '收起侧边栏'}
        >
          {collapsed ? <ChevronRight className="size-4" /> : <ChevronLeft className="size-4" />}
        </button>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto">
        {/* Top bar with command palette trigger */}
        <div className="sticky top-0 z-10 flex h-12 items-center justify-end gap-2 border-b bg-background/95 px-6 backdrop-blur">
          <span className="text-xs text-muted-foreground">当前用户：{getUsername()}</span>
          <button
            type="button"
            onClick={() => setPaletteOpen(true)}
            className="flex items-center gap-2 rounded-md border bg-muted/40 px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-muted"
            aria-label="打开命令面板"
          >
            <Search className="size-3.5" />
            <span>搜索服务 / 主机…</span>
            <kbd className="ml-2 rounded border bg-background px-1.5 py-0.5 font-mono text-[10px]">Ctrl K</kbd>
          </button>
          <button
            type="button"
            onClick={handleLogout}
            className="flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <LogOut className="size-3.5" />
            退出
          </button>
        </div>
        <div className="mx-auto max-w-7xl p-6">
          <Outlet />
        </div>
      </main>

      <Toaster position="top-center" richColors />
      <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} />
    </div>
  )
}
