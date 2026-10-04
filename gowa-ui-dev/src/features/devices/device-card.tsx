import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  CircleUserRound,
  KeyRound,
  MoreVertical,
  QrCode,
  RefreshCw,
  Trash2,
  Unplug,
  Webhook,
  CheckCircle,
} from 'lucide-react'
import { toast } from 'sonner'
import { logoutDevice, reconnectDevice, removeDevice } from '@/api/devices'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { StateBadge } from '@/features/devices/state-badge'
import { DeviceWebhookDialog } from '@/features/devices/webhook-dialog'
import { useDeviceAvatar } from '@/hooks/use-device-avatar'
import { toApiError } from '@/lib/api-error'
import { formatDate } from '@/lib/format'
import { cn } from '@/lib/utils'
import { useDeviceStore } from '@/stores/device'
import type { RegistryDevice } from '@/api/types'

export function DeviceCard({
  device,
  onLoginQr,
  onLoginCode,
}: {
  device: RegistryDevice
  onLoginQr: (device: RegistryDevice) => void
  onLoginCode: (device: RegistryDevice) => void
}) {
  const queryClient = useQueryClient()
  const selectedDeviceId = useDeviceStore((state) => state.selectedDeviceId)
  const selectDevice = useDeviceStore((state) => state.selectDevice)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [webhookOpen, setWebhookOpen] = useState(false)
  const selected = selectedDeviceId === device.id
  const avatar = useDeviceAvatar(device)

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ['devices'] })

  const logout = useMutation({
    mutationFn: () => logoutDevice(device.id),
    onSuccess: () => {
      toast.success(`Logout requested for ${device.id}`)
      void invalidate()
    },
    onError: (error) => toast.error(toApiError(error).message),
  })

  const reconnect = useMutation({
    mutationFn: () => reconnectDevice(device.id),
    onSuccess: () => {
      toast.success(`Reconnect requested for ${device.id}`)
      void invalidate()
    },
    onError: (error) => toast.error(toApiError(error).message),
  })

  const remove = useMutation({
    mutationFn: () => removeDevice(device.id),
    onSuccess: () => {
      toast.success(`Device ${device.id} removed`)
      if (selected) selectDevice(null)
      void invalidate()
    },
    onError: (error) => toast.error(toApiError(error).message),
  })

  return (
    <Card
      className={cn(
        'card-lift glass-card gap-3.5 rounded-xl transition-all duration-200',
        selected &&
          'border-primary/50 bg-primary/[0.04] ring-1 ring-primary/30 shadow-md shadow-primary/5',
      )}
    >
      <CardHeader className="flex flex-row items-start justify-between gap-2 space-y-0 pb-1">
        <div className="flex min-w-0 items-center gap-3">
          <div className="relative">
            <Avatar size="lg" className="border border-border/80 shadow-2xs">
              {avatar.data?.url && (
                <AvatarImage src={avatar.data.url} alt={device.display_name || device.id} />
              )}
              <AvatarFallback className="bg-muted/50">
                <CircleUserRound className="size-5 text-muted-foreground" />
              </AvatarFallback>
            </Avatar>
            {device.state === 'logged_in' && (
              <span className="absolute -bottom-0.5 -right-0.5 size-3 rounded-full border-2 border-background bg-emerald-500" />
            )}
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-1.5">
              <p className="truncate text-sm font-semibold text-foreground">
                {device.display_name || device.id}
              </p>
              {selected && (
                <span className="flex items-center gap-0.5 rounded border border-primary/30 bg-primary/10 px-1 py-0.2 text-[9px] font-semibold text-primary">
                  <CheckCircle className="size-2.5" />
                  Active
                </span>
              )}
            </div>
            <p className="truncate font-mono text-[11px] text-muted-foreground">
              {device.jid || device.phone_number || 'Unpaired session'}
            </p>
          </div>
        </div>
        <StateBadge state={device.state} />
      </CardHeader>

      <CardContent className="space-y-1 text-[11px] text-muted-foreground">
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground/80">Device ID:</span>
          <span className="rounded bg-muted/60 px-1.5 py-0.5 font-mono text-[10px] text-foreground">
            {device.id}
          </span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground/80">Registered:</span>
          <span>{formatDate(device.created_at)}</span>
        </div>
      </CardContent>

      <CardFooter className="flex items-center justify-between gap-2 pt-2">
        <Button
          variant={selected ? 'default' : 'outline'}
          size="sm"
          onClick={() => selectDevice(device.id)}
          disabled={selected}
          className="h-7.5 text-xs font-medium"
        >
          {selected ? 'Active Scope' : 'Select'}
        </Button>

        <div className="flex items-center gap-1">
          {device.state !== 'logged_in' && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => onLoginQr(device)}
              className="h-7.5 gap-1.5 border-primary/30 bg-primary/10 text-xs font-medium text-primary hover:bg-primary/20"
            >
              <QrCode className="size-3.5" />
              Pair
            </Button>
          )}

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label="Device actions"
                className="h-7.5 w-7.5"
              >
                <MoreVertical className="size-3.5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="text-xs">
              <DropdownMenuItem onClick={() => onLoginQr(device)}>
                <QrCode className="size-3.5 text-primary" /> Login with QR
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onLoginCode(device)}>
                <KeyRound className="size-3.5" /> Login with Pairing Code
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => reconnect.mutate()}>
                <RefreshCw className="size-3.5" /> Reconnect Session
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => logout.mutate()}>
                <Unplug className="size-3.5" /> Logout Device
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setWebhookOpen(true)}>
                <Webhook className="size-3.5 text-primary" /> Webhook Setup
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                variant="destructive"
                onClick={() => setConfirmDelete(true)}
                className="text-destructive"
              >
                <Trash2 className="size-3.5" /> Delete Slot
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardFooter>

      <DeviceWebhookDialog device={device} open={webhookOpen} onOpenChange={setWebhookOpen} />

      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent className="glass-card">
          <AlertDialogHeader>
            <AlertDialogTitle>Delete device {device.id}?</AlertDialogTitle>
            <AlertDialogDescription className="text-xs">
              This will permanently revoke the device slot and clear its WhatsApp credentials from GOWA.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="h-8 text-xs">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => remove.mutate()}
              className="h-8 bg-destructive text-xs text-white hover:bg-destructive/90"
            >
              Delete Slot
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  )
}
