import { cn } from '@/lib/utils'

/** Mono-styled identifier (phone, JID, message ID, group ID). */
export function IdText({ value, className }: { value: string; className?: string }) {
  return (
    <span className={cn('text-muted-foreground font-mono text-xs break-all', className)}>
      {value}
    </span>
  )
}
