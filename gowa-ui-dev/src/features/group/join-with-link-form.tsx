import { useState, type FormEvent } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { joinGroupWithLink } from '@/api/group'
import { ResultPanel } from '@/components/shared/result-panel'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useActionMutation } from '@/hooks/use-action-mutation'

export function JoinWithLinkForm() {
  const queryClient = useQueryClient()
  const [link, setLink] = useState('')

  const mutation = useActionMutation(joinGroupWithLink, {
    successMessage: 'Joined group',
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['groups'] }),
  })

  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    mutation.mutate({ link })
  }

  return (
    <form className="flex flex-col gap-4" onSubmit={onSubmit}>
      <div className="flex flex-col gap-2">
        <Label htmlFor="join-link">Invite link</Label>
        <Input
          id="join-link"
          placeholder="https://chat.whatsapp.com/xxxxxxxx"
          value={link}
          onChange={(event) => setLink(event.target.value)}
          required
        />
      </div>
      <Button type="submit" disabled={mutation.isPending} className="self-start">
        {mutation.isPending && <Loader2 className="size-4 animate-spin" />}
        Join group
      </Button>
      <ResultPanel result={mutation.data} />
    </form>
  )
}
