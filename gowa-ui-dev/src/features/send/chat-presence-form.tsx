import { useState, type FormEvent } from 'react'
import { chatPresenceRequest, sendChatPresence } from '@/api/send'
import { FormActions } from '@/components/shared/curl-dialog'
import { ResultPanel } from '@/components/shared/result-panel'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useActionMutation } from '@/hooks/use-action-mutation'
import { useRecipientJid } from '@/stores/recipient'

export function SendChatPresenceForm() {
  const jid = useRecipientJid()
  const [action, setAction] = useState('start')

  const mutation = useActionMutation(sendChatPresence, { successMessage: 'Chat presence sent' })

  const payload = {
    phone: jid,
    action,
  }

  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    mutation.mutate(payload)
  }

  return (
    <form className="flex flex-col gap-4" onSubmit={onSubmit}>
      <div className="flex flex-col gap-2">
        <Label>Action</Label>
        <Select value={action} onValueChange={setAction}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="start">Start typing</SelectItem>
            <SelectItem value="stop">Stop typing</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <FormActions
        submitLabel="Send chat presence"
        pending={mutation.isPending}
        disabled={!jid}
        request={chatPresenceRequest(payload)}
      />
      <ResultPanel result={mutation.data} />
    </form>
  )
}
