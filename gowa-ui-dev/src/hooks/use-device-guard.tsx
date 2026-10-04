import { Smartphone } from 'lucide-react'
import { EmptyState } from '@/components/shared/empty-state'
import { useDeviceStore } from '@/stores/device'

/** Renders a banner when no device is selected; feature pages gate on this. */
export function DeviceGuard() {
  return (
    <EmptyState
      icon={Smartphone}
      title="No device selected"
      hint="Select a device from the top bar to use device-scoped actions."
    />
  )
}

export function useSelectedDevice(): string | null {
  return useDeviceStore((state) => state.selectedDeviceId)
}
