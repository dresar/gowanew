import { useEffect } from 'react'
import { Smartphone } from 'lucide-react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useDevices } from '@/hooks/use-devices'
import { cn } from '@/lib/utils'
import { useDeviceStore } from '@/stores/device'
import type { DeviceState } from '@/api/types'

const stateDots: Record<DeviceState, string> = {
  logged_in: 'bg-emerald-500 shadow-xs shadow-emerald-500/50',
  connected: 'bg-sky-500 shadow-xs shadow-sky-500/50',
  connecting: 'bg-amber-500 animate-pulse',
  disconnected: 'bg-muted-foreground/40',
}

export function DeviceSwitcher() {
  const { data: devices } = useDevices()
  const selectedDeviceId = useDeviceStore((state) => state.selectedDeviceId)
  const selectDevice = useDeviceStore((state) => state.selectDevice)

  useEffect(() => {
    if (!devices) return
    const exists = devices.some((device) => device.id === selectedDeviceId)
    if (!exists) selectDevice(devices[0]?.id ?? null)
  }, [devices, selectedDeviceId, selectDevice])

  if (!devices?.length) return null

  return (
    <Select value={selectedDeviceId ?? undefined} onValueChange={selectDevice}>
      <SelectTrigger
        size="sm"
        className="h-7.5 w-36 rounded-lg border-border/70 bg-card/60 text-xs backdrop-blur-md transition-all hover:border-primary/40 sm:w-44 md:w-56"
      >
        <Smartphone className="size-3.5 shrink-0 text-primary" />
        <SelectValue placeholder="Select device" />
      </SelectTrigger>
      <SelectContent className="border-border/80 bg-popover/95 backdrop-blur-xl">
        {devices.map((device) => (
          <SelectItem key={device.id} value={device.id} className="text-xs">
            <span className={cn('size-2 shrink-0 rounded-full', stateDots[device.state])} />
            <span className="truncate">{device.display_name || device.id}</span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
