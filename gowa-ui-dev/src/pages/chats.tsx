import { useRef, useState } from 'react'
import { MessagesSquare } from 'lucide-react'
import { ChatList } from '@/features/chat/chat-list'
import { MessageView } from '@/features/chat/message-view'
import { Card } from '@/components/ui/card'
import { PageHeader } from '@/components/shared/page-header'
import { selectedChatForDevice, type ChatSelection } from '@/features/chat/device-scope'
import { DeviceGuard, useSelectedDevice } from '@/hooks/use-device-guard'
import type { ChatInfo } from '@/api/chat'

export default function ChatsPage() {
  const device = useSelectedDevice()
  const [selection, setSelection] = useState<ChatSelection | null>(null)
  const messagePane = useRef<HTMLDivElement>(null)
  const selected = selectedChatForDevice(selection, device)

  const handleSelect = (chat: ChatInfo) => {
    if (!device) return
    setSelection({ deviceId: device, chat })
    messagePane.current?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
  }

  if (!device) {
    return (
      <div className="flex flex-col gap-4">
        <PageHeader title="Chat History" description="Live stored WhatsApp conversations for this device." />
        <DeviceGuard />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-3 sm:gap-4 lg:h-[calc(100svh-8rem)]">
      <PageHeader title="Chat History" description="Live stored WhatsApp conversations for this device." />
      <div className="grid gap-3 sm:gap-4 lg:min-h-0 lg:flex-1 lg:grid-cols-[330px_1fr]">
        <Card className="glass-card h-[24rem] overflow-hidden rounded-xl p-3 backdrop-blur-xl lg:h-auto lg:min-h-0">
          <ChatList
            key={device}
            deviceId={device}
            selectedJid={selected?.jid ?? null}
            onSelect={handleSelect}
          />
        </Card>
        <Card
          ref={messagePane}
          className="glass-card h-[calc(100svh-9rem)] min-h-[26rem] overflow-hidden rounded-xl p-3 backdrop-blur-xl lg:h-auto lg:min-h-0"
        >
          {selected ? (
            <MessageView key={`${device}:${selected.jid}`} chat={selected} deviceId={device} />
          ) : (
            <div className="flex h-full flex-col items-center justify-center gap-2.5 text-muted-foreground">
              <div className="flex size-12 items-center justify-center rounded-xl border border-border/80 bg-muted/40 text-primary shadow-2xs">
                <MessagesSquare className="size-6" />
              </div>
              <p className="text-xs font-medium text-foreground">Select a conversation</p>
              <p className="text-[11px] text-muted-foreground">
                Click any chat from the left panel to stream stored messages
              </p>
            </div>
          )}
        </Card>
      </div>
    </div>
  )
}
