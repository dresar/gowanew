import { http, results } from '@/lib/http'

export function rejectCall(payload: { caller_jid: string; call_id: string }) {
  return results<{ status: string }>(http.post('/call/reject', payload))
}
