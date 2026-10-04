import { useState, type FormEvent } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { changePushName } from '@/api/user'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useActionMutation } from '@/hooks/use-action-mutation'

export function PushnameForm() {
  const [pushName, setPushName] = useState('')
  const queryClient = useQueryClient()

  const mutation = useActionMutation(changePushName, {
    successMessage: 'Push name updated',
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['devices'] })
    },
  })

  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    mutation.mutate(pushName)
  }

  return (
    <form className="flex flex-col gap-4" onSubmit={onSubmit}>
      <div className="flex flex-col gap-2">
        <Label htmlFor="push-name">New push name</Label>
        <Input
          id="push-name"
          value={pushName}
          onChange={(event) => setPushName(event.target.value)}
          placeholder="Your display name"
          required
        />
      </div>
      <Button
        type="submit"
        disabled={mutation.isPending || !pushName.trim()}
        className="self-start"
      >
        {mutation.isPending && <Loader2 className="size-4 animate-spin" />}
        Update push name
      </Button>
    </form>
  )
}
