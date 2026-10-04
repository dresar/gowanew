import logoUrl from '@/assets/gowa-logo.webp'
import { cn } from '@/lib/utils'

export function Logo({ className }: { className?: string }) {
  return (
    <div className={cn('flex items-center gap-2.5', className)}>
      <div className="relative flex size-8 shrink-0 items-center justify-center rounded-lg border border-emerald-500/25 bg-emerald-500/10 p-1 shadow-2xs backdrop-blur-xs">
        <img src={logoUrl} alt="gowa logo" className="size-6 object-contain" />
      </div>
      <div className="flex flex-col">
        <div className="flex items-center gap-1.5">
          <span className="font-heading text-sm font-bold tracking-tight text-foreground">
            GOWA
          </span>
          <span className="rounded border border-primary/25 bg-primary/15 px-1 py-0.5 font-mono text-[9px] font-semibold text-primary">
            API
          </span>
        </div>
        <span className="text-muted-foreground text-[10px] leading-none">WhatsApp Multi-Device</span>
      </div>
    </div>
  )
}
