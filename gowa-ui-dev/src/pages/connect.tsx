import { useEffect, useState, type FormEvent } from 'react'
import { CheckCircle2, Globe, KeyRound, Loader2, Lock, ShieldCheck, User } from 'lucide-react'
import { Navigate, useNavigate } from 'react-router-dom'
import { Logo } from '@/components/layout/logo'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useConnection, type TestResult } from '@/stores/connection'

const errorMessages: Record<Exclude<TestResult, 'ok'>, string> = {
  unauthorized: 'The server rejected these credentials (401 Unauthorized).',
  'not-gowa': 'That URL answered, but it does not appear to be an active GOWA server.',
  unreachable: 'Could not reach the server. Make sure GOWA backend is running on that port.',
}

export default function ConnectPage() {
  const navigate = useNavigate()
  const status = useConnection((state) => state.status)
  const storedUrl = useConnection((state) => state.baseUrl)
  const storedUser = useConnection((state) => state.username)
  const connect = useConnection((state) => state.connect)

  const [url, setUrl] = useState(
    storedUrl ?? (import.meta.env.VITE_DEFAULT_SERVER_URL as string | undefined) ?? 'http://localhost:3000',
  )
  const [username, setUsername] = useState(storedUser ?? '')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (url && (status === 'unconfigured' || status === 'booting')) {
      let active = true
      const tryAutoConnect = async () => {
        setSubmitting(true)
        const result = await connect(url, username || undefined, password || undefined)
        if (active) {
          setSubmitting(false)
          if (result === 'ok') {
            navigate('/', { replace: true })
          }
        }
      }
      void tryAutoConnect()
      return () => {
        active = false
      }
    }
  }, [url, status, username, password, connect, navigate])

  if (status === 'connected') return <Navigate to="/" replace />

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault()
    setSubmitting(true)
    setError(null)
    const result = await connect(url, username || undefined, password || undefined)
    setSubmitting(false)
    if (result === 'ok') {
      navigate('/', { replace: true })
    } else {
      setError(errorMessages[result])
    }
  }

  const selectPreset = (presetUrl: string) => {
    setUrl(presetUrl)
  }

  return (
    <div className="ambient-glow relative flex min-h-svh items-center justify-center overflow-hidden bg-background p-4 sm:p-6">
      {/* Background ambient lighting */}
      <div
        aria-hidden
        className="pointer-events-none absolute -top-40 left-1/2 -translate-x-1/2 h-96 w-full max-w-4xl bg-[radial-gradient(ellipse_at_center,oklch(0.72_0.18_155/15%),transparent_70%)] blur-2xl"
      />

      <Card className="glass-card relative w-full max-w-md border-border/70 shadow-2xl backdrop-blur-2xl">
        <CardHeader className="gap-2 pb-4">
          <div className="flex items-center justify-between">
            <Logo className="[&_img]:size-8" />
            <div className="flex items-center gap-1 rounded-md border border-emerald-500/25 bg-emerald-500/10 px-2 py-0.5 text-[10px] font-medium text-emerald-600 dark:text-emerald-400">
              <ShieldCheck className="size-3" />
              <span>TLS / LocalStorage</span>
            </div>
          </div>
          <div>
            <CardTitle className="text-lg">Connect to GOWA Server</CardTitle>
            <CardDescription className="text-xs">
              Direct connection to your WhatsApp API instance. Credentials stay strictly in your browser.
            </CardDescription>
          </div>
        </CardHeader>
        <CardContent>
          <form className="flex flex-col gap-4" onSubmit={onSubmit}>
            {/* Server URL Input with quick presets */}
            <div className="flex flex-col gap-1.5">
              <div className="flex items-center justify-between">
                <Label htmlFor="server-url" className="text-xs font-medium">
                  Server Endpoint URL
                </Label>
                <div className="flex gap-1">
                  <button
                    type="button"
                    onClick={() => selectPreset('http://localhost:3000')}
                    className="rounded bg-muted/60 px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground hover:bg-primary/15 hover:text-primary transition-colors"
                  >
                    :3000
                  </button>
                  <button
                    type="button"
                    onClick={() => selectPreset('http://localhost:5173/gowa')}
                    className="rounded bg-muted/60 px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground hover:bg-primary/15 hover:text-primary transition-colors"
                  >
                    /gowa
                  </button>
                </div>
              </div>
              <div className="relative flex items-center">
                <Globe className="absolute left-2.5 size-3.5 text-muted-foreground" />
                <Input
                  id="server-url"
                  placeholder="http://localhost:3000"
                  value={url}
                  onChange={(event) => setUrl(event.target.value)}
                  className="pl-8 text-xs font-mono"
                  required
                />
              </div>
            </div>

            {/* Optional Basic Auth */}
            <div className="grid grid-cols-2 gap-3">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="username" className="text-xs font-medium">
                  Username (optional)
                </Label>
                <div className="relative flex items-center">
                  <User className="absolute left-2.5 size-3.5 text-muted-foreground" />
                  <Input
                    id="username"
                    autoComplete="username"
                    placeholder="user"
                    value={username}
                    onChange={(event) => setUsername(event.target.value)}
                    className="pl-8 text-xs"
                  />
                </div>
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="password" className="text-xs font-medium">
                  Password (optional)
                </Label>
                <div className="relative flex items-center">
                  <KeyRound className="absolute left-2.5 size-3.5 text-muted-foreground" />
                  <Input
                    id="password"
                    type="password"
                    autoComplete="current-password"
                    placeholder="••••••"
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                    className="pl-8 text-xs font-mono"
                  />
                </div>
              </div>
            </div>

            {/* Feedback messages */}
            {error && (
              <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-2.5 text-xs text-destructive">
                {error}
              </div>
            )}
            {status === 'unauthorized' && !error && (
              <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-2.5 text-xs text-destructive">
                The stored credentials were rejected — please enter them again.
              </div>
            )}
            {status === 'unreachable' && !error && (
              <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 p-2.5 text-xs text-amber-600 dark:text-amber-400">
                Server unreachable — verify GOWA backend is started on that port.
              </div>
            )}

            <Button
              type="submit"
              disabled={submitting || !url.trim()}
              className="mt-1 h-9 w-full gap-2 text-xs font-semibold shadow-md shadow-primary/20"
            >
              {submitting ? (
                <>
                  <Loader2 className="size-3.5 animate-spin" />
                  <span>Connecting to Gateway…</span>
                </>
              ) : (
                <>
                  <CheckCircle2 className="size-3.5" />
                  <span>Connect to Gateway</span>
                </>
              )}
            </Button>

            <div className="flex items-center justify-center gap-1.5 pt-1 text-[11px] text-muted-foreground/80">
              <Lock className="size-3 text-emerald-500" />
              <span>Token-based secure handshake</span>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
