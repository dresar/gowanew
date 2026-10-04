import { useLayoutEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import { keepPreviousData, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2, Send } from 'lucide-react'
import { getChatMessages, type ChatInfo, type MessageInfo } from '@/api/chat'
import { sendText } from '@/api/send'
import { MessageMedia } from '@/features/chat/message-media'
import { ChatControls } from '@/features/chat/chat-controls'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Switch } from '@/components/ui/switch'
import { chatMessagesQueryKey } from '@/features/chat/device-scope'
import { chatDisplayName, senderDisplayName } from '@/features/chat/display-name'
import { useActionMutation } from '@/hooks/use-action-mutation'
import { formatDate } from '@/lib/format'
import { cn } from '@/lib/utils'
import { ScheduleFields } from '@/features/send/schedule-fields'
import { useScheduleDraft } from '@/features/send/use-schedule-draft'

const PAGE_SIZE = 30

function sortMessagesChronologically(messages: MessageInfo[]): MessageInfo[] {
  return [...messages].sort((left, right) => {
    return new Date(left.timestamp).getTime() - new Date(right.timestamp).getTime()
  })
}

function dayKey(timestamp: string): string {
  return new Date(timestamp).toDateString()
}

function MessageBubble({ message, deviceId }: { message: MessageInfo; deviceId: string }) {
  const hasMedia = message.media_type && message.media_type !== ''
  return (
    <div className={cn('flex', message.is_from_me ? 'justify-end' : 'justify-start')}>
      <div
        className={cn(
          'max-w-[82%] rounded-xl px-3 py-2 text-xs sm:text-[13px] shadow-2xs backdrop-blur-md transition-all',
          message.is_from_me
            ? 'rounded-tr-xs border border-emerald-500/25 bg-emerald-500/15 text-foreground'
            : 'rounded-tl-xs border border-border/80 bg-card/85 text-foreground',
        )}
      >
        {!message.is_from_me && (
          <p className="mb-0.5 font-mono text-[10px] font-semibold text-primary">
            {senderDisplayName(message)}
          </p>
        )}
        {message.content && <p className="break-words whitespace-pre-wrap leading-relaxed">{message.content}</p>}
        {hasMedia && <MessageMedia message={message} deviceId={deviceId} />}
        {message.reactions && message.reactions.length > 0 && (
          <div className="mt-1 flex flex-wrap gap-1">
            {message.reactions.map((r) => (
              <span key={r.emoji} className="rounded-md border border-border/60 bg-muted/60 px-1 py-0.2 text-[10px]">
                {r.emoji}
              </span>
            ))}
          </div>
        )}
        <p className="mt-1 text-right font-mono text-[9px] text-muted-foreground/80">
          {formatDate(message.timestamp)}
        </p>
      </div>
    </div>
  )
}

export function MessageView({ chat, deviceId }: { chat: ChatInfo; deviceId: string }) {
  const queryClient = useQueryClient()
  const messageList = useRef<HTMLDivElement>(null)
  const [search, setSearch] = useState('')
  const [mediaOnly, setMediaOnly] = useState(false)
  const [offset, setOffset] = useState(0)
  const [draft, setDraft] = useState('')
  const { draft: scheduleDraft, patch: patchSchedule, reset: resetSchedule } = useScheduleDraft()

  const query = useQuery({
    queryKey: chatMessagesQueryKey(deviceId, chat.jid, { search, mediaOnly, offset }),
    queryFn: () =>
      getChatMessages(
        chat.jid,
        {
          search: search || undefined,
          media_only: mediaOnly || undefined,
          limit: PAGE_SIZE,
          offset,
        },
        deviceId,
      ),
    placeholderData: keepPreviousData,
  })

  const messages = useMemo(
    () => sortMessagesChronologically(query.data?.data ?? []),
    [query.data?.data],
  )
  const total = query.data?.pagination.total ?? 0
  const responseChat = query.data?.chat_info
  const resolvedChat = responseChat?.jid === chat.jid ? responseChat : chat

  useLayoutEffect(() => {
    const viewport = messageList.current?.querySelector<HTMLElement>(
      '[data-slot="scroll-area-viewport"]',
    )
    if (viewport) viewport.scrollTop = viewport.scrollHeight
  }, [chat.jid, messages])

  const sendMutation = useActionMutation(
    (message: string) => sendText({ phone: chat.jid, message, ...scheduleDraft }),
    {
      successMessage: (r) => (r.schedule_id ? r.status : 'Message sent'),
      onSuccess: () => {
        setDraft('')
        resetSchedule()
        void queryClient.invalidateQueries({
          queryKey: ['chat-messages', deviceId, chat.jid],
        })
      },
    },
  )

  const onSend = (event: FormEvent) => {
    event.preventDefault()
    if (draft.trim()) sendMutation.mutate(draft.trim())
  }

  return (
    <div className="flex h-full flex-col gap-3">
      {/* Header bar for selected chat */}
      <div className="flex items-center justify-between border-b border-border/60 pb-2.5">
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold text-foreground">
            {chatDisplayName(resolvedChat)}
          </p>
          <p className="truncate font-mono text-[10px] text-muted-foreground">{resolvedChat.jid}</p>
        </div>
        <ChatControls chat={resolvedChat} />
      </div>

      {/* Filter toolbar */}
      <div className="flex items-center justify-between gap-2">
        <Input
          className="h-7.5 max-w-xs text-xs"
          placeholder="Filter messages in this chat…"
          value={search}
          onChange={(event) => {
            setSearch(event.target.value)
            setOffset(0)
          }}
        />
        <label className="flex items-center gap-1.5 text-xs text-muted-foreground select-none">
          <Switch
            checked={mediaOnly}
            onCheckedChange={(value) => {
              setMediaOnly(value)
              setOffset(0)
            }}
          />
          <span className="hidden sm:inline">Media only</span>
        </label>
      </div>

      {/* Message stream */}
      <div ref={messageList} className="min-h-0 flex-1">
        <ScrollArea className="size-full rounded-xl border border-border/60 bg-muted/20 p-3 backdrop-blur-xs">
          {query.isLoading ? (
            <div className="flex justify-center p-6">
              <Loader2 className="size-5 animate-spin text-primary" />
            </div>
          ) : messages.length === 0 ? (
            <div className="flex flex-col gap-1 p-6 text-center text-xs text-muted-foreground">
              <p className="font-medium text-foreground">No messages stored for this chat yet.</p>
              <p className="text-[11px]">
                Messages will appear as sent/received or when history sync completes.
              </p>
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              {messages.map((message, index) => {
                const showDateSeparator =
                  index === 0 || dayKey(message.timestamp) !== dayKey(messages[index - 1].timestamp)
                return (
                  <div key={message.id}>
                    {showDateSeparator && (
                      <div className="flex justify-center py-1">
                        <span className="rounded-md border border-border/60 bg-card/85 px-2.5 py-0.5 font-mono text-[10px] text-muted-foreground shadow-2xs backdrop-blur-xs">
                          {new Date(message.timestamp).toLocaleDateString(undefined, {
                            day: 'numeric',
                            month: 'short',
                            year: 'numeric',
                          })}
                        </span>
                      </div>
                    )}
                    <MessageBubble message={message} deviceId={deviceId} />
                  </div>
                )
              })}
            </div>
          )}
        </ScrollArea>
      </div>

      {/* Pagination toolbar */}
      <div className="flex items-center justify-between text-[11px] text-muted-foreground">
        <span>{total} messages stored</span>
        <div className="flex gap-1">
          <Button
            variant="outline"
            size="xs"
            disabled={offset === 0}
            onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
            className="h-6.5 text-[11px]"
          >
            Newer
          </Button>
          <Button
            variant="outline"
            size="xs"
            disabled={offset + PAGE_SIZE >= total}
            onClick={() => setOffset(offset + PAGE_SIZE)}
            className="h-6.5 text-[11px]"
          >
            Older
          </Button>
        </div>
      </div>

      {/* Interactive composer */}
      <form className="flex flex-col gap-2" onSubmit={onSend}>
        <ScheduleFields draft={scheduleDraft} patch={patchSchedule} />
        <div className="flex gap-2">
          <Input
            placeholder="Type a message to dispatch…"
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            className="h-8.5 text-xs"
          />
          <Button
            type="submit"
            disabled={sendMutation.isPending || !draft.trim()}
            className="h-8.5 gap-1.5 px-3.5 text-xs font-semibold"
          >
            {sendMutation.isPending ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <Send className="size-3.5" />
            )}
            Send
          </Button>
        </div>
      </form>
    </div>
  )
}
