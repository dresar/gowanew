import * as React from 'react'

import { cn } from '@/lib/utils'

function Input({ className, type, ...props }: React.ComponentProps<'input'>) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        'border-border/70 bg-card/60 backdrop-blur-md file:text-foreground placeholder:text-muted-foreground/60 focus-visible:border-primary/70 focus-visible:ring-primary/25 disabled:bg-muted/40 aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:bg-card/40 dark:disabled:bg-input/80 dark:aria-invalid:border-destructive/50 dark:aria-invalid:ring-destructive/40 h-8.5 w-full min-w-0 rounded-lg border px-3 py-1 text-xs sm:text-sm transition-all outline-none file:inline-flex file:h-6 file:border-0 file:bg-transparent file:text-xs file:font-medium focus-visible:ring-2 disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:ring-2 shadow-2xs',
        className,
      )}
      {...props}
    />
  )
}

export { Input }
