import { useTheme } from 'next-themes'
import { CheckCircle2, Globe, LogOut, Moon, Server, Sun, Laptop } from 'lucide-react'
import { PageHeader } from '@/components/shared/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { useAppInfo } from '@/hooks/use-app-info'
import { formatBytes } from '@/lib/format'
import { useConnection } from '@/stores/connection'

export default function SettingsPage() {
  const baseUrl = useConnection((state) => state.baseUrl)
  const username = useConnection((state) => state.username)
  const disconnect = useConnection((state) => state.disconnect)
  const { data: info, isLoading: infoLoading, error: infoError } = useAppInfo()
  const { theme, setTheme } = useTheme()

  return (
    <div className="flex max-w-2xl flex-col gap-4">
      <PageHeader
        title="Settings & Diagnostics"
        description="Active gateway connectivity, server telemetry, and visual preferences."
      />

      {/* Connection Card */}
      <Card className="glass-card rounded-xl backdrop-blur-xl">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2 text-sm font-semibold">
              <Globe className="size-4 text-primary" />
              Gateway Connection
            </CardTitle>
            <span className="flex items-center gap-1 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-[10px] font-medium text-emerald-500">
              <CheckCircle2 className="size-3" />
              Connected
            </span>
          </div>
          <CardDescription className="text-xs">
            Where this dashboard dispatches REST and WebSocket events.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 text-xs">
          <div className="flex items-center justify-between gap-4 rounded-lg border border-border/50 bg-muted/30 p-2.5">
            <span className="font-medium text-muted-foreground">Server Origin</span>
            <span className="truncate font-mono text-foreground">{baseUrl}</span>
          </div>
          <div className="flex items-center justify-between gap-4 rounded-lg border border-border/50 bg-muted/30 p-2.5">
            <span className="font-medium text-muted-foreground">Auth Principal</span>
            <span className="font-mono text-foreground">{username || '— (No basic auth)'}</span>
          </div>
          <div className="pt-1">
            <Button
              variant="outline"
              size="sm"
              onClick={disconnect}
              className="h-8 gap-1.5 border-destructive/30 text-destructive hover:bg-destructive/10"
            >
              <LogOut className="size-3.5" />
              Disconnect Session
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Server Capabilities */}
      <Card className="glass-card rounded-xl backdrop-blur-xl">
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-sm font-semibold">
            <Server className="size-4 text-primary" />
            Server Architecture
          </CardTitle>
          <CardDescription className="text-xs">Payload telemetry reported by GET /app/info</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-2.5 text-xs">
          {infoLoading && <Skeleton className="h-20 rounded-lg" />}
          {infoError && (
            <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-destructive">
              Failed to query server info.
            </div>
          )}
          {info && (
            <div className="grid gap-2 sm:grid-cols-2">
              <div className="rounded-lg border border-border/50 bg-muted/30 p-2.5">
                <span className="text-[10px] text-muted-foreground">Version</span>
                <p className="font-mono font-semibold text-foreground">{info.version}</p>
              </div>
              <div className="rounded-lg border border-border/50 bg-muted/30 p-2.5">
                <span className="text-[10px] text-muted-foreground">OS Identity</span>
                <p className="font-mono font-semibold text-foreground">{info.os}</p>
              </div>
              <div className="rounded-lg border border-border/50 bg-muted/30 p-2.5">
                <span className="text-[10px] text-muted-foreground">Max Upload Size</span>
                <p className="font-mono font-semibold text-foreground">{formatBytes(info.max_video_size)}</p>
              </div>
              <div className="rounded-lg border border-border/50 bg-muted/30 p-2.5">
                <span className="text-[10px] text-muted-foreground">Scheduled Sends</span>
                <p className="font-semibold text-emerald-500">
                  {info.scheduled_sends ? 'Enabled' : 'Disabled'}
                </p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Visual Appearance */}
      <Card className="glass-card rounded-xl backdrop-blur-xl">
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-sm font-semibold">
            <Sun className="size-4 text-primary" />
            Visual Appearance
          </CardTitle>
          <CardDescription className="text-xs">
            Toggle between Glassmorphic Dark and Light mode themes.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex max-w-xs flex-col gap-1.5">
            <Select value={theme} onValueChange={setTheme}>
              <SelectTrigger className="h-8.5 text-xs">
                <SelectValue placeholder="Theme" />
              </SelectTrigger>
              <SelectContent className="text-xs">
                <SelectItem value="dark">
                  <div className="flex items-center gap-2">
                    <Moon className="size-3.5" />
                    <span>Dark (Obsidian Glass)</span>
                  </div>
                </SelectItem>
                <SelectItem value="light">
                  <div className="flex items-center gap-2">
                    <Sun className="size-3.5" />
                    <span>Light (Crystalline Frost)</span>
                  </div>
                </SelectItem>
                <SelectItem value="system">
                  <div className="flex items-center gap-2">
                    <Laptop className="size-3.5" />
                    <span>System Sync</span>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
