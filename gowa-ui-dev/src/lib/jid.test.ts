import { describe, expect, it } from 'vitest'
import { composeJid, isStatus } from './jid'

describe('composeJid', () => {
  it('appends the user suffix', () => {
    expect(composeJid('628123', 'user')).toBe('628123@s.whatsapp.net')
  })

  it('appends the group suffix', () => {
    expect(composeJid('12345', 'group')).toBe('12345@g.us')
  })

  it('appends newsletter and lid suffixes', () => {
    expect(composeJid('abc', 'newsletter')).toBe('abc@newsletter')
    expect(composeJid('99', 'lid')).toBe('99@lid')
  })

  it('returns the status broadcast jid regardless of phone', () => {
    expect(composeJid('anything', 'status')).toBe('status@broadcast')
    expect(composeJid('', 'status')).toBe('status@broadcast')
  })

  it('passes through a value that already contains a jid', () => {
    expect(composeJid('12345@g.us', 'user')).toBe('12345@g.us')
  })

  it('returns empty for a blank phone (non-status)', () => {
    expect(composeJid('   ', 'user')).toBe('')
  })
})

describe('isStatus', () => {
  it('is true only for status', () => {
    expect(isStatus('status')).toBe(true)
    expect(isStatus('user')).toBe(false)
  })
})
