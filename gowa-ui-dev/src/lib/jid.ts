export const JID_TYPES = {
  user: '@s.whatsapp.net',
  group: '@g.us',
  newsletter: '@newsletter',
  lid: '@lid',
  status: 'status@broadcast',
} as const

export type RecipientType = keyof typeof JID_TYPES

export const recipientOptions: { value: RecipientType; label: string }[] = [
  { value: 'user', label: 'Private message' },
  { value: 'group', label: 'Group' },
  { value: 'newsletter', label: 'Newsletter' },
  { value: 'lid', label: 'LID (Linked ID)' },
  { value: 'status', label: 'Status' },
]

/** Compose the WhatsApp JID the API expects from a phone/id and recipient type. */
export function composeJid(phone: string, type: RecipientType): string {
  if (type === 'status') return JID_TYPES.status
  const trimmed = phone.trim()
  if (!trimmed) return ''
  if (trimmed.includes('@')) return trimmed
  return `${trimmed}${JID_TYPES[type]}`
}

export function isStatus(type: RecipientType): boolean {
  return type === 'status'
}
