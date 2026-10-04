import { type FormEvent } from 'react'
import { Check, Loader2, X } from 'lucide-react'
import {
  approveParticipantRequests,
  listParticipantRequests,
  rejectParticipantRequests,
} from '@/api/group'
import { Button } from '@/components/ui/button'
import { useActionMutation } from '@/hooks/use-action-mutation'
import { formatDate } from '@/lib/format'

/** Strip the @server suffix from a JID for compact display. */
function shortId(jid: string): string {
  return jid.split('@')[0]
}

export function ParticipantRequests({ groupJid }: { groupJid: string }) {
  const load = useActionMutation(listParticipantRequests, {
    successMessage: 'Loaded requests',
  })

  const decide = useActionMutation(
    ({ action, jid }: { action: 'approve' | 'reject'; jid: string }) =>
      (action === 'approve' ? approveParticipantRequests : rejectParticipantRequests)({
        group_id: groupJid,
        participants: [jid],
      }),
    {
      successMessage: 'Request handled',
      onSuccess: () => load.mutate({ group_id: groupJid }),
    },
  )

  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    load.mutate({ group_id: groupJid })
  }

  const requests = load.data
  const pendingJid = decide.isPending ? decide.variables?.jid : undefined

  return (
    <div className="flex flex-col gap-4">
      <form className="flex flex-col gap-4" onSubmit={onSubmit}>
        <Button type="submit" disabled={load.isPending} className="self-start">
          {load.isPending && <Loader2 className="size-4 animate-spin" />}
          Load requests
        </Button>
      </form>

      {requests && requests.length === 0 && (
        <p className="text-muted-foreground text-sm">No pending requests.</p>
      )}

      {requests && requests.length > 0 && (
        <ul className="flex flex-col gap-2">
          {requests.map((request) => (
            <li
              key={request.jid}
              className="flex items-center justify-between gap-2 rounded-md border p-3"
            >
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">
                  {request.display_name || request.phone_number || shortId(request.jid)}
                </p>
                <p className="text-muted-foreground truncate font-mono text-xs">
                  {shortId(request.jid)} · {formatDate(request.requested_at)}
                </p>
              </div>
              <div className="flex items-center gap-1">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={decide.isPending}
                  onClick={() => decide.mutate({ action: 'approve', jid: request.jid })}
                >
                  {pendingJid === request.jid ? (
                    <Loader2 className="size-4 animate-spin" />
                  ) : (
                    <Check className="size-4" />
                  )}
                  Approve
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  className="text-destructive hover:text-destructive"
                  disabled={decide.isPending}
                  onClick={() => decide.mutate({ action: 'reject', jid: request.jid })}
                >
                  <X className="size-4" />
                  Reject
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
