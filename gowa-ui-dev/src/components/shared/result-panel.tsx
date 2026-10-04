import { CheckCircle2 } from 'lucide-react'

export function ResultPanel({ result }: { result: unknown }) {
  if (result === undefined || result === null) return null
  return (
    <div className="animate-in fade-in slide-in-from-bottom-1 border-live/30 bg-live/5 flex flex-col gap-2 rounded-lg border p-3 duration-300">
      <div className="text-primary flex items-center gap-2 text-sm font-medium">
        <CheckCircle2 className="text-live size-4" />
        Success
      </div>
      <pre className="text-muted-foreground overflow-x-auto font-mono text-xs">
        {JSON.stringify(result, null, 2)}
      </pre>
    </div>
  )
}
