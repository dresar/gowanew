import { beforeEach, describe, expect, it } from 'vitest'
import { useRecipientStore } from '@/stores/recipient'

describe('recipient store', () => {
  beforeEach(() => {
    useRecipientStore.setState({
      recipient: { phone: '', type: 'user' },
      recents: [],
    })
  })

  it('sets the current recipient', () => {
    useRecipientStore.getState().setRecipient({ phone: '628123', type: 'group' })
    expect(useRecipientStore.getState().recipient).toEqual({ phone: '628123', type: 'group' })
  })

  it('pushes a recent to the front and dedupes', () => {
    const { pushRecent } = useRecipientStore.getState()
    pushRecent({ phone: '1', type: 'user' })
    pushRecent({ phone: '2', type: 'user' })
    pushRecent({ phone: '1', type: 'user' })
    expect(useRecipientStore.getState().recents).toEqual([
      { phone: '1', type: 'user' },
      { phone: '2', type: 'user' },
    ])
  })

  it('treats same phone with different type as distinct recents', () => {
    const { pushRecent } = useRecipientStore.getState()
    pushRecent({ phone: '1', type: 'user' })
    pushRecent({ phone: '1', type: 'group' })
    expect(useRecipientStore.getState().recents).toHaveLength(2)
  })

  it('ignores empty phone', () => {
    useRecipientStore.getState().pushRecent({ phone: '   ', type: 'user' })
    expect(useRecipientStore.getState().recents).toEqual([])
  })

  it('caps recents at 8', () => {
    const { pushRecent } = useRecipientStore.getState()
    for (let i = 1; i <= 10; i++) pushRecent({ phone: String(i), type: 'user' })
    const recents = useRecipientStore.getState().recents
    expect(recents).toHaveLength(8)
    expect(recents[0]).toEqual({ phone: '10', type: 'user' })
  })
})
