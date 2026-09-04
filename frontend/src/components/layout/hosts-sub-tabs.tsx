import { NavLink } from 'react-router-dom'
import { Server, Activity, Container, Copy } from 'lucide-react'
import { cn } from '@/lib/utils'

const SUB_TABS = [
  { to: '/hosts', label: '主机列表', icon: Server },
  { to: '/monitor', label: '资源监控', icon: Activity },
  { to: '/containers', label: '容器管理', icon: Container },
  { to: '/hosts/image-sync', label: '镜像同步', icon: Copy },
] as const

/**
 * 主机管理 / 资源监控 共用顶部子 Tab。
 * 两条路由互为入口，一处入口在侧边栏「基础设施」组下。
 */
export function HostsSubTabs() {
  return (
    <div className="flex items-center gap-1 border-b">
      {SUB_TABS.map(({ to, label, icon: Icon }) => (
        <NavLink
          key={to}
          to={to}
          end
          className={({ isActive }) =>
            cn(
              'flex items-center gap-1.5 px-3 py-2 text-sm font-medium border-b-2 -mb-px transition-colors',
              isActive
                ? 'border-primary text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            )
          }
        >
          <Icon className="size-4" />
          {label}
        </NavLink>
      ))}
    </div>
  )
}
