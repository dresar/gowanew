import { useState } from 'react'
import { Eye, Link, Plus } from 'lucide-react'
import { PageHeader } from '@/components/shared/page-header'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { DeviceGuard, useSelectedDevice } from '@/hooks/use-device-guard'
import { CreateGroupForm } from '@/features/group/create-group-form'
import { GroupDetailSheet } from '@/features/group/group-detail-sheet'
import { GroupDirectory } from '@/features/group/group-list'
import { InfoFromLinkForm } from '@/features/group/info-from-link-form'
import { JoinWithLinkForm } from '@/features/group/join-with-link-form'
import type { MyGroup } from '@/api/group'

function HeaderDialog({
  trigger,
  title,
  description,
  children,
}: {
  trigger: React.ReactNode
  title: string
  description: string
  children: React.ReactNode
}) {
  return (
    <Dialog>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        {children}
      </DialogContent>
    </Dialog>
  )
}

export default function GroupsPage() {
  const deviceId = useSelectedDevice()
  const [selected, setSelected] = useState<MyGroup | null>(null)

  return (
    <>
      <PageHeader
        title="Groups"
        description="Groups this device belongs to — select one to manage it."
        actions={
          <>
            <HeaderDialog
              trigger={
                <Button variant="outline" size="sm">
                  <Eye className="size-4" />
                  Preview link
                </Button>
              }
              title="Group info from link"
              description="Preview a group before joining."
            >
              <InfoFromLinkForm />
            </HeaderDialog>
            <HeaderDialog
              trigger={
                <Button variant="outline" size="sm">
                  <Link className="size-4" />
                  Join with link
                </Button>
              }
              title="Join with link"
              description="Join a group from its invite link."
            >
              <JoinWithLinkForm />
            </HeaderDialog>
            <HeaderDialog
              trigger={
                <Button size="sm">
                  <Plus className="size-4" />
                  Create group
                </Button>
              }
              title="Create group"
              description="Start a new group with participants."
            >
              <CreateGroupForm />
            </HeaderDialog>
          </>
        }
      />
      {!deviceId ? (
        <DeviceGuard />
      ) : (
        <>
          <GroupDirectory onSelect={setSelected} />
          <GroupDetailSheet group={selected} onOpenChange={(open) => !open && setSelected(null)} />
        </>
      )}
    </>
  )
}
