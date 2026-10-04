import { useState, type FormEvent } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { createGroup } from '@/api/group'
import { ResultPanel } from '@/components/shared/result-panel'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { useActionMutation } from '@/hooks/use-action-mutation'

/** Split a textarea into a trimmed list, one entry per line or comma. */
function parseList(text: string): string[] {
  return text
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

export function CreateGroupForm() {
  const queryClient = useQueryClient()
  const [title, setTitle] = useState('')
  const [participants, setParticipants] = useState('')

  const mutation = useActionMutation(createGroup, {
    successMessage: 'Group created',
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['groups'] }),
  })

  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    mutation.mutate({ title, participants: parseList(participants) })
  }

  return (
    <form className="flex flex-col gap-4" onSubmit={onSubmit}>
      <div className="flex flex-col gap-2">
        <Label htmlFor="create-group-title">Group name</Label>
        <Input
          id="create-group-title"
          value={title}
          onChange={(event) => setTitle(event.target.value)}
          required
        />
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="create-group-participants">Participants</Label>
        <Textarea
          id="create-group-participants"
          rows={4}
          placeholder="628xxxxxxxxxx, one per line or comma-separated"
          value={participants}
          onChange={(event) => setParticipants(event.target.value)}
        />
      </div>
      <Button type="submit" disabled={mutation.isPending} className="self-start">
        {mutation.isPending && <Loader2 className="size-4 animate-spin" />}
        Create group
      </Button>
      <ResultPanel result={mutation.data} />
    </form>
  )
}
