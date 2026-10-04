import { BellRing, PhoneOff } from 'lucide-react'
import { ActionCard } from '@/components/shared/action-card'
import { PageHeader } from '@/components/shared/page-header'
import { CallRejectForm } from '@/features/call/call-reject-form'
import { NewsletterList } from '@/features/newsletter/newsletter-list'
import { DeviceGuard, useSelectedDevice } from '@/hooks/use-device-guard'

export default function MiscPage() {
  const device = useSelectedDevice()

  return (
    <div className="flex flex-col gap-4 sm:gap-5">
      <PageHeader
        title="Channels & Telephony"
        description="WhatsApp channel / newsletter subscriptions and incoming call routing."
      />

      {!device ? (
        <DeviceGuard />
      ) : (
        <div className="grid items-start gap-4 lg:grid-cols-2">
          <ActionCard
            icon={BellRing}
            title="Channel Broadcasts"
            description="Newsletters and public updates followed by this WhatsApp device."
          >
            <NewsletterList />
          </ActionCard>
          <ActionCard
            icon={PhoneOff}
            title="Reject Incoming Call"
            description="Reject a voice/video call using caller JID and Call ID payload from webhooks."
          >
            <CallRejectForm />
          </ActionCard>
        </div>
      )}
    </div>
  )
}
