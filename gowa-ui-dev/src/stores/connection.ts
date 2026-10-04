import axios from 'axios'
import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import type { ResponseData } from '@/api/types'
import { basicAuthHeader } from '@/lib/api-error'
import { normalizeBaseUrl, sameOriginBaseUrl } from '@/lib/url'

export type ConnectionStatus =
  'booting' | 'unconfigured' | 'connected' | 'unauthorized' | 'unreachable'

export type TestResult = 'ok' | 'unauthorized' | 'not-gowa' | 'unreachable'

export interface ConnectionState {
  baseUrl: string | null
  username: string | null
  password: string | null
  status: ConnectionStatus
  connect: (baseUrl: string, username?: string, password?: string) => Promise<TestResult>
  boot: () => Promise<void>
  disconnect: () => void
  markUnauthorized: () => void
}

/**
 * Probe a server without the shared axios instance (no interceptors, no
 * global 401 handling). Distinguishes a real gowa server from any web server
 * that happens to answer 200 (e.g. an SPA dev server echoing index.html).
 */
export async function probeServer(
  baseUrl: string,
  username?: string,
  password?: string,
): Promise<TestResult> {
  try {
    const response = await axios.get<ResponseData<unknown>>(`${baseUrl}/devices`, {
      timeout: 5_000,
      validateStatus: () => true,
      headers: {
        Accept: 'application/json',
        ...(username && password ? { Authorization: basicAuthHeader(username, password) } : {}),
      },
    })
    if (response.status === 401) return 'unauthorized'
    const body = response.data
    if (response.status === 200 && typeof body === 'object' && body !== null && 'code' in body) {
      return 'ok'
    }
    return 'not-gowa'
  } catch {
    return 'unreachable'
  }
}

export const useConnection = create<ConnectionState>()(
  persist(
    (set, get) => ({
      baseUrl: null,
      username: null,
      password: null,
      status: 'booting',

      connect: async (rawUrl, username, password) => {
        const baseUrl = normalizeBaseUrl(rawUrl)
        const result = await probeServer(baseUrl, username, password)
        if (result === 'ok') {
          set({
            baseUrl,
            username: username || null,
            password: password || null,
            status: 'connected',
          })
        }
        return result
      },

      boot: async () => {
        const { baseUrl, username, password } = get()
        if (baseUrl) {
          const stored = await probeServer(baseUrl, username ?? undefined, password ?? undefined)
          if (stored === 'ok') {
            set({ status: 'connected' })
            return
          }
          if (stored === 'unauthorized') {
            set({ status: 'unauthorized' })
            return
          }
        }

        // 1. Same-origin check: when served directly by GOWA at http://localhost:3000/
        const origin = sameOriginBaseUrl()
        if ((await probeServer(origin)) === 'ok') {
          set({ baseUrl: origin, status: 'connected' })
          return
        }

        // 2. Default backend URL check (e.g. http://localhost:3000)
        const defaultServer = normalizeBaseUrl(
          (import.meta.env.VITE_DEFAULT_SERVER_URL as string | undefined) || 'http://localhost:3000',
        )
        if (defaultServer && defaultServer !== origin) {
          if ((await probeServer(defaultServer)) === 'ok') {
            set({ baseUrl: defaultServer, status: 'connected' })
            return
          }
        }

        // 3. /gowa dev proxy fallback for local Vite dev
        if (origin.includes(':5173')) {
          const gowaProxy = `${origin}/gowa`
          if ((await probeServer(gowaProxy)) === 'ok') {
            set({ baseUrl: gowaProxy, status: 'connected' })
            return
          }
        }

        set({ status: 'unconfigured' })
      },

      disconnect: () =>
        set({ baseUrl: null, username: null, password: null, status: 'unconfigured' }),

      markUnauthorized: () => {
        if (get().status === 'connected') set({ status: 'unauthorized' })
      },
    }),
    {
      name: 'gowa-ui.connection.v1',
      storage: createJSONStorage(() => localStorage),
      partialize: ({ baseUrl, username, password }) => ({ baseUrl, username, password }),
    },
  ),
)
