import { CircleUserRound } from 'lucide-react'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useSelectedDevice } from '@/hooks/use-device-guard'
import { useDeviceAvatar } from '@/hooks/use-device-avatar'
import { useDevices } from '@/hooks/use-devices'

export function MyProfileCard() {
  const deviceId = useSelectedDevice()
  const { data: devices } = useDevices()
  const device = devices?.find((item) => item.id === deviceId)
  const avatar = useDeviceAvatar(device)

  return (
    <Card>
      <CardContent className="flex items-center gap-4">
        <Avatar className="size-16">
          {avatar.data?.url && <AvatarImage src={avatar.data.url} alt="My avatar" />}
          <AvatarFallback>
            {avatar.isLoading ? (
              <Skeleton className="size-full rounded-full" />
            ) : (
              <CircleUserRound className="text-muted-foreground size-8" />
            )}
          </AvatarFallback>
        </Avatar>
        <div className="flex min-w-0 flex-col gap-0.5">
          <p className="truncate text-base font-semibold">
            {device?.display_name || 'No push name set'}
          </p>
          <p className="text-muted-foreground truncate text-sm">
            {device?.phone_number || device?.jid || 'not paired yet'}
          </p>
        </div>
      </CardContent>
    </Card>
  )
}
