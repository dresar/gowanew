import { useState, type FormEvent } from 'react'
import { Loader2 } from 'lucide-react'
import { rejectCall } from '@/api/call'
import { ResultPanel } from '@/components/shared/result-panel'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useActionMutation } from '@/hooks/use-action-mutation'

export function CallRejectForm() {
  const [callerJid, setCallerJid] = useState('')
  const [callId, setCallId] = useState('')
  const mutation = useActionMutation(rejectCall, { successMessage: 'Call rejected' })

  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    mutation.mutate({ caller_jid: callerJid.trim(), call_id: callId.trim() })
  }

  return (
    <form className="flex flex-col gap-4" onSubmit={onSubmit}>
      <p className="text-muted-foreground text-sm">
        Reject an incoming call using the caller JID and call ID from the call webhook event.
      </p>
      <div className="flex flex-col gap-2">
        <Label htmlFor="caller-jid">Caller JID</Label>
        <Input
          id="caller-jid"
          placeholder="628xxxxxxxxxx@s.whatsapp.net"
          value={callerJid}
          onChange={(event) => setCallerJid(event.target.value)}
          required
        />
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="call-id">Call ID</Label>
        <Input
          id="call-id"
          placeholder="Call ID from the webhook event"
          value={callId}
          onChange={(event) => setCallId(event.target.value)}
          required
        />
      </div>
      <Button
        type="submit"
        disabled={mutation.isPending || !callerJid.trim() || !callId.trim()}
        className="self-start"
      >
        {mutation.isPending && <Loader2 className="size-4 animate-spin" />}
        Reject call
      </Button>
      <ResultPanel result={mutation.data} />
    </form>
  )
}
