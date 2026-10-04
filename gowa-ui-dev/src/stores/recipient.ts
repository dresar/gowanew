import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { composeJid, type RecipientType } from '@/lib/jid'

export interface Recipient {
  phone: string
  type: RecipientType
}

interface RecipientState {
  recipient: Recipient
  recents: Recipient[]
  setRecipient: (recipient: Recipient) => void
  pushRecent: (recipient: Recipient) => void
}

const MAX_RECENTS = 8

/**
 * Shared recipient context for the Messaging workspace: entered once, reused
 * by every compose/act form. Recent recipients persist across sessions.
 */
export const useRecipientStore = create<RecipientState>()(
  persist(
    (set) => ({
      recipient: { phone: '', type: 'user' },
      recents: [],
      setRecipient: (recipient) => set({ recipient }),
      pushRecent: (recipient) =>
        set((state) => {
          if (!recipient.phone.trim()) return state
          const recents = [
            recipient,
            ...state.recents.filter(
              (item) => !(item.phone === recipient.phone && item.type === recipient.type),
            ),
          ].slice(0, MAX_RECENTS)
          return { recents }
        }),
    }),
    {
      name: 'gowa-recipient',
      partialize: (state) => ({ recents: state.recents }),
    },
  ),
)

/** Composed JID for the current recipient ('' when incomplete). */
export function useRecipientJid(): string {
  return useRecipientStore((state) => composeJid(state.recipient.phone, state.recipient.type))
}
