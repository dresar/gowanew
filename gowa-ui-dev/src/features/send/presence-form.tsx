import { useState, type FormEvent } from 'react'
import { presenceRequest, sendPresence } from '@/api/send'
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

export function SendPresenceForm() {
  const [type, setType] = useState('available')

  const mutation = useActionMutation(sendPresence, { successMessage: 'Presence updated' })

  const payload = { type }

  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    mutation.mutate(payload)
  }

  return (
    <form className="flex flex-col gap-4" onSubmit={onSubmit}>
      <div className="flex flex-col gap-2">
        <Label>Presence</Label>
        <Select value={type} onValueChange={setType}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="available">Available</SelectItem>
            <SelectItem value="unavailable">Unavailable</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <FormActions
        submitLabel="Update presence"
        pending={mutation.isPending}
        disabled={false}
        request={presenceRequest(payload)}
      />
      <ResultPanel result={mutation.data} />
    </form>
  )
}
