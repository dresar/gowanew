import { useId } from 'react'
import { composeJid, isStatus, recipientOptions, type RecipientType } from '@/lib/jid'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

export interface RecipientValue {
  phone: string
  type: RecipientType
}

export function RecipientField({
  value,
  onChange,
  showStatus = false,
}: {
  value: RecipientValue
  onChange: (value: RecipientValue) => void
  showStatus?: boolean
}) {
  const options = recipientOptions.filter((option) => showStatus || option.value !== 'status')
  const jid = composeJid(value.phone, value.type)
  const id = useId()

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2">
        <Label>Recipient type</Label>
        <Select
          value={value.type}
          onValueChange={(type) => onChange({ ...value, type: type as RecipientType })}
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {options.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      {!isStatus(value.type) && (
        <div className="flex flex-col gap-2">
          <Label htmlFor={id}>Phone / Group ID</Label>
          <Input
            id={id}
            placeholder="628xxxxxxxxxx"
            value={value.phone}
            onChange={(event) => onChange({ ...value, phone: event.target.value })}
          />
          {jid && <p className="text-muted-foreground font-mono text-xs">{jid}</p>}
        </div>
      )}
    </div>
  )
}
