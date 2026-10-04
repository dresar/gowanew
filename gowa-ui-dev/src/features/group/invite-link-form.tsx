import { useState, type FormEvent } from 'react'
import { Loader2 } from 'lucide-react'
import { getGroupInviteLink } from '@/api/group'
import { ResultPanel } from '@/components/shared/result-panel'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { useActionMutation } from '@/hooks/use-action-mutation'

export function InviteLinkForm({ groupJid }: { groupJid: string }) {
  const [reset, setReset] = useState(false)

  const mutation = useActionMutation(getGroupInviteLink, {
    successMessage: 'Loaded invite link',
  })

  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    mutation.mutate({ group_id: groupJid, reset })
  }

  return (
    <form className="flex flex-col gap-4" onSubmit={onSubmit}>
      <label className="flex items-center gap-2 text-sm">
        <Switch checked={reset} onCheckedChange={setReset} />
        Reset link (revokes the previous one)
      </label>
      <Button type="submit" disabled={mutation.isPending} className="self-start">
        {mutation.isPending && <Loader2 className="size-4 animate-spin" />}
        Get invite link
      </Button>
      <ResultPanel result={mutation.data} />
    </form>
  )
}
