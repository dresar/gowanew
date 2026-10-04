import { useState } from 'react'
import {
  LayoutDashboard,
  Loader2,
  Menu,
  MessagesSquare,
  Send,
  Settings,
  UserRound,
  Users,
  Wrench,
  CalendarClock,
} from 'lucide-react'
import { Navigate, NavLink, Outlet, useLocation } from 'react-router-dom'
import { DeviceSwitcher } from '@/components/layout/device-switcher'
import { Logo } from '@/components/layout/logo'
import { ThemeToggle } from '@/components/layout/theme-toggle'
import { WsBadge } from '@/components/layout/ws-badge'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { PasskeyDialog } from '@/features/session/passkey-dialog'
import { useAppInfo } from '@/hooks/use-app-info'
import { cn } from '@/lib/utils'
import { useConnection } from '@/stores/connection'

const navGroups = [
  {
    label: 'Overview',
    items: [{ to: '/', label: 'Devices', icon: LayoutDashboard }],
  },
  {
    label: 'Messaging',
    items: [
      { to: '/messaging', label: 'Messaging', icon: Send },
      { to: '/scheduled', label: 'Scheduled', icon: CalendarClock },
      { to: '/chats', label: 'Chats', icon: MessagesSquare },
    ],
  },
  {
    label: 'Directory',
    items: [
      { to: '/groups', label: 'Groups', icon: Users },
      { to: '/account', label: 'Account', icon: UserRound },
    ],
  },
  {
    label: 'System',
    items: [
      { to: '/misc', label: 'Channels & Calls', icon: Wrench },
      { to: '/settings', label: 'Settings', icon: Settings },
    ],
  },
]

function NavContent({ onNavigate }: { onNavigate?: () => void }) {
  const { data: info } = useAppInfo()
  return (
    <nav className="flex flex-col gap-4">
      {navGroups.map((group) => (
        <div key={group.label} className="flex flex-col gap-0.5">
          <p className="px-3 pb-1 text-[10px] font-semibold tracking-wider text-muted-foreground/70 uppercase">
            {group.label}
          </p>
          {group.items
            .filter(({ to }) => to !== '/scheduled' || info?.scheduled_sends)
            .map(({ to, label, icon: Icon }) => (
              <NavLink
                key={to}
                to={to}
                end={to === '/'}
                onClick={onNavigate}
                className={({ isActive }) =>
                  cn(
                    'group/nav relative flex items-center gap-2.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-all duration-150',
                    isActive
                      ? 'border border-primary/20 bg-primary/12 font-semibold text-primary shadow-2xs'
                      : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground',
                  )
                }
              >
                {({ isActive }) => (
                  <>
                    <Icon
                      className={cn(
                        'size-4 shrink-0 transition-colors',
                        isActive
                          ? 'text-primary'
                          : 'text-muted-foreground group-hover/nav:text-foreground',
                      )}
                    />
                    <span className="truncate">{label}</span>
                    {isActive && (
                      <span className="ml-auto size-1.5 rounded-full bg-primary shadow-xs shadow-primary/80" />
                    )}
                  </>
                )}
              </NavLink>
            ))}
        </div>
      ))}
    </nav>
  )
}

export function AppShell() {
  const status = useConnection((state) => state.status)
  const location = useLocation()
  const [mobileNavOpen, setMobileNavOpen] = useState(false)

  if (status === 'booting') {
    return (
      <div className="ambient-glow flex min-h-svh items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="size-6 animate-spin text-primary" />
          <p className="text-xs text-muted-foreground">Connecting to GOWA session…</p>
        </div>
      </div>
    )
  }

  if (status !== 'connected') {
    return <Navigate to="/connect" replace />
  }

  return (
    <div className="ambient-glow relative flex min-h-svh bg-background/95">
      {/* Desktop Glass Sidebar */}
      <aside className="hidden w-58 shrink-0 flex-col border-r border-sidebar-border/70 bg-sidebar/70 backdrop-blur-xl md:flex">
        <div className="flex h-13 items-center border-b border-sidebar-border/70 px-4">
          <Logo />
        </div>
        <ScrollArea className="flex-1 px-2.5 py-3">
          <NavContent />
        </ScrollArea>
        <div className="border-t border-sidebar-border/70 p-3">
          <div className="flex items-center justify-between rounded-lg border border-border/40 bg-card/40 px-2.5 py-1.5 text-[11px] text-muted-foreground backdrop-blur-xs">
            <span>Core v9.6</span>
            <span className="font-mono text-emerald-500">PureGo</span>
          </div>
        </div>
      </aside>

      {/* Mobile Drawer */}
      <Sheet open={mobileNavOpen} onOpenChange={setMobileNavOpen}>
        <SheetContent
          side="left"
          className="w-72 border-r border-sidebar-border/70 bg-sidebar/95 p-0 backdrop-blur-xl"
        >
          <SheetHeader className="border-b border-sidebar-border/70 p-4">
            <SheetTitle asChild>
              <div>
                <Logo />
              </div>
            </SheetTitle>
          </SheetHeader>
          <ScrollArea className="flex-1 px-3 py-4">
            <NavContent onNavigate={() => setMobileNavOpen(false)} />
          </ScrollArea>
        </SheetContent>
      </Sheet>

      {/* Main Content Area */}
      <div className="flex min-w-0 flex-1 flex-col">
        {/* Sticky Glass Navbar */}
        <header className="sticky top-0 z-30 flex h-13 items-center justify-between gap-2 border-b border-border/60 bg-background/75 px-4 backdrop-blur-xl sm:px-6">
          <div className="flex items-center gap-2 md:hidden">
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label="Open navigation"
              onClick={() => setMobileNavOpen(true)}
            >
              <Menu className="size-4" />
            </Button>
            <Logo />
          </div>
          <div className="ml-auto flex items-center gap-2 sm:gap-2.5">
            <DeviceSwitcher />
            <WsBadge />
            <ThemeToggle />
          </div>
        </header>

        {/* Dynamic Page Router */}
        <main className="flex-1 p-3.5 sm:p-5 md:p-6">
          <div key={location.pathname} className="stagger mx-auto flex max-w-5xl flex-col gap-4 sm:gap-5">
            <Outlet />
          </div>
        </main>
      </div>
      <PasskeyDialog />
    </div>
  )
}
