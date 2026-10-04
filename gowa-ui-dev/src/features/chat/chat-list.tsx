import { useState } from 'react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Loader2, Search } from 'lucide-react'
import { listChats, type ChatInfo } from '@/api/chat'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Switch } from '@/components/ui/switch'
import { chatListQueryKey } from '@/features/chat/device-scope'
import { chatDisplayName } from '@/features/chat/display-name'
import { formatDate, isZeroTime } from '@/lib/format'
import { cn } from '@/lib/utils'

const PAGE_SIZE = 25

export function ChatList({
  deviceId,
  selectedJid,
  onSelect,
}: {
  deviceId: string
  selectedJid: string | null
  onSelect: (chat: ChatInfo) => void
}) {
  const [search, setSearch] = useState('')
  const [hasMedia, setHasMedia] = useState(false)
  const [offset, setOffset] = useState(0)

  const query = useQuery({
    queryKey: chatListQueryKey(deviceId, { search, hasMedia, offset }),
    queryFn: () =>
      listChats(
        {
          search: search || undefined,
          has_media: hasMedia || undefined,
          limit: PAGE_SIZE,
          offset,
        },
        deviceId,
      ),
    placeholderData: keepPreviousData,
  })

  const chats = query.data?.data ?? []
  const total = query.data?.pagination.total ?? 0

  return (
    <div className="flex h-full flex-col gap-2.5">
      <div className="flex flex-col gap-2">
        <div className="relative">
          <Search className="absolute left-2.5 top-2.5 size-3.5 text-muted-foreground" />
          <Input
            className="h-8 pl-8 text-xs font-medium"
            placeholder="Search conversations…"
            value={search}
            onChange={(event) => {
              setSearch(event.target.value)
              setOffset(0)
            }}
          />
        </div>
        <label className="flex items-center gap-2 text-xs text-muted-foreground select-none">
          <Switch
            checked={hasMedia}
            onCheckedChange={(value) => {
              setHasMedia(value)
              setOffset(0)
            }}
          />
          <span>Media attachments only</span>
        </label>
      </div>

      <ScrollArea className="min-h-0 flex-1 rounded-lg border border-border/60 bg-card/40 backdrop-blur-md">
        {query.isLoading ? (
          <div className="flex justify-center p-6">
            <Loader2 className="size-5 animate-spin text-primary" />
          </div>
        ) : chats.length === 0 ? (
          <p className="p-6 text-center text-xs text-muted-foreground">No conversations found</p>
        ) : (
          <ul className="divide-y divide-border/40">
            {chats.map((chat) => (
              <li key={chat.jid}>
                <button
                  type="button"
                  onClick={() => onSelect(chat)}
                  className={cn(
                    'group/item flex w-full items-center gap-2.5 px-3 py-2 text-left transition-all duration-150',
                    selectedJid === chat.jid
                      ? 'border-l-2 border-primary bg-primary/12 font-medium text-foreground'
                      : 'hover:bg-muted/40 text-muted-foreground hover:text-foreground',
                  )}
                >
                  <span className="flex size-8 shrink-0 items-center justify-center rounded-lg border border-border/70 bg-card/80 font-heading text-xs font-bold text-foreground shadow-2xs">
                    {chatDisplayName(chat).slice(0, 1).toUpperCase()}
                  </span>
                  <span className="flex min-w-0 flex-col">
                    <span className="truncate text-xs font-semibold text-foreground">
                      {chatDisplayName(chat)}
                    </span>
                    <span className="truncate font-mono text-[10px] text-muted-foreground">
                      {isZeroTime(chat.last_message_time)
                        ? chat.jid
                        : formatDate(chat.last_message_time)}
                    </span>
                  </span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </ScrollArea>

      <div className="flex items-center justify-between text-[11px] text-muted-foreground">
        <span>{total} chats listed</span>
        <div className="flex gap-1">
          <Button
            variant="outline"
            size="xs"
            disabled={offset === 0}
            onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
            className="h-6.5 text-[11px]"
          >
            Prev
          </Button>
          <Button
            variant="outline"
            size="xs"
            disabled={offset + PAGE_SIZE >= total}
            onClick={() => setOffset(offset + PAGE_SIZE)}
            className="h-6.5 text-[11px]"
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  )
}
