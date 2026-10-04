import { useState } from 'react'
import {
  CheckCircle2,
  Radio,
  Smartphone,
  Unplug,
} from 'lucide-react'
import { EmptyState } from '@/components/shared/empty-state'
import { PageHeader } from '@/components/shared/page-header'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { CreateDeviceDialog } from '@/features/devices/create-device-dialog'
import { DeviceCard } from '@/features/devices/device-card'
import { LoginCodeDialog } from '@/features/session/login-code-dialog'
import { LoginQrDialog } from '@/features/session/login-qr-dialog'
import { useDevices } from '@/hooks/use-devices'
import { toApiError } from '@/lib/api-error'
import type { RegistryDevice } from '@/api/types'

export default function DashboardPage() {
  const { data: devices, isLoading, error } = useDevices()
  const [qrDevice, setQrDevice] = useState<RegistryDevice | null>(null)
  const [codeDevice, setCodeDevice] = useState<RegistryDevice | null>(null)

  const total = devices?.length ?? 0
  const loggedIn = devices?.filter((d) => d.state === 'logged_in').length ?? 0
  const connecting = devices?.filter((d) => d.state === 'connecting' || d.state === 'connected').length ?? 0
  const disconnected = devices?.filter((d) => d.state === 'disconnected').length ?? 0

  return (
    <div className="flex flex-col gap-4 sm:gap-5">
      <PageHeader
        title="WhatsApp Devices"
        description="Multi-device session orchestration and telemetry."
        actions={<CreateDeviceDialog />}
      />

      {/* Glass Telemetry / Stats Cards */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 sm:gap-4">
        <div className="glass-card flex items-center gap-3 rounded-xl p-3.5 backdrop-blur-xl">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-border/80 bg-card/60 text-primary shadow-2xs">
            <Smartphone className="size-4" />
          </div>
          <div className="min-w-0">
            <p className="text-[11px] font-medium text-muted-foreground">Total Slots</p>
            <p className="font-heading text-lg font-bold tracking-tight text-foreground">{total}</p>
          </div>
        </div>

        <div className="glass-card flex items-center gap-3 rounded-xl p-3.5 backdrop-blur-xl">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-emerald-500/30 bg-emerald-500/10 text-emerald-500 shadow-2xs">
            <CheckCircle2 className="size-4" />
          </div>
          <div className="min-w-0">
            <p className="text-[11px] font-medium text-muted-foreground">Logged In</p>
            <p className="font-heading text-lg font-bold tracking-tight text-emerald-500">{loggedIn}</p>
          </div>
        </div>

        <div className="glass-card flex items-center gap-3 rounded-xl p-3.5 backdrop-blur-xl">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-amber-500/30 bg-amber-500/10 text-amber-500 shadow-2xs">
            <Radio className="size-4 animate-pulse" />
          </div>
          <div className="min-w-0">
            <p className="text-[11px] font-medium text-muted-foreground">Connecting</p>
            <p className="font-heading text-lg font-bold tracking-tight text-amber-500">{connecting}</p>
          </div>
        </div>

        <div className="glass-card flex items-center gap-3 rounded-xl p-3.5 backdrop-blur-xl">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-border/60 bg-muted/40 text-muted-foreground shadow-2xs">
            <Unplug className="size-4" />
          </div>
          <div className="min-w-0">
            <p className="text-[11px] font-medium text-muted-foreground">Offline</p>
            <p className="font-heading text-lg font-bold tracking-tight text-muted-foreground">{disconnected}</p>
          </div>
        </div>
      </div>

      {error && (
        <Card className="border-destructive/40 bg-destructive/10">
          <CardContent className="py-3 text-xs text-destructive">
            Failed to load devices: {toApiError(error).message}
          </CardContent>
        </Card>
      )}

      {isLoading && (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <Skeleton className="h-44 rounded-xl" />
          <Skeleton className="h-44 rounded-xl" />
          <Skeleton className="h-44 rounded-xl" />
        </div>
      )}

      {devices && devices.length === 0 && (
        <EmptyState
          icon={Smartphone}
          title="No WhatsApp devices registered"
          hint="Create a new device slot above, then link your phone via QR scan or WhatsApp pairing code."
        />
      )}

      {devices && devices.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {devices.map((device) => (
            <DeviceCard
              key={device.id}
              device={device}
              onLoginQr={setQrDevice}
              onLoginCode={setCodeDevice}
            />
          ))}
        </div>
      )}

      <LoginQrDialog device={qrDevice} onOpenChange={(open) => !open && setQrDevice(null)} />
      <LoginCodeDialog device={codeDevice} onOpenChange={(open) => !open && setCodeDevice(null)} />
    </div>
  )
}
