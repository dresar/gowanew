import type { ReactNode, ComponentType } from 'react'
import { Card, CardContent } from '@/components/ui/card'

export function EmptyState({
  icon: Icon,
  title,
  hint,
  action,
}: {
  icon: ComponentType<{ className?: string }>
  title: string
  hint?: string
  action?: ReactNode
}) {
  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-3 py-12 text-center">
        <div className="bg-accent text-accent-foreground flex size-12 items-center justify-center rounded-full">
          <Icon className="size-6" />
        </div>
        <div className="flex flex-col gap-1">
          <p className="font-heading font-medium">{title}</p>
          {hint && <p className="text-muted-foreground max-w-sm text-sm">{hint}</p>}
        </div>
        {action}
      </CardContent>
    </Card>
  )
}
