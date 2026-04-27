import { NavLink } from 'react-router-dom'
import { Activity, Users, Radio, BarChart3, BookOpen, Boxes } from 'lucide-react'
import { cn } from '@/lib/utils'

const links = [
  { to: '/', label: 'Overview', icon: Activity },
  { to: '/subscribers', label: 'Subscribers', icon: Users },
  { to: '/events', label: 'Event Stream', icon: Radio },
  { to: '/topics', label: 'Topics', icon: BookOpen },
  { to: '/services', label: 'Services', icon: Boxes },
  { to: '/metrics', label: 'Metrics', icon: BarChart3 },
]

export function Navigation() {
  return (
    <nav className="flex flex-col gap-1 px-3">
      {links.map(({ to, label, icon: Icon }) => (
        <NavLink
          key={to}
          to={to}
          end={to === '/'}
          className={({ isActive }) =>
            cn(
              'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
              isActive
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
            )
          }
        >
          <Icon className="h-4 w-4" />
          {label}
        </NavLink>
      ))}
    </nav>
  )
}
