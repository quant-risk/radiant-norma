'use client'

/**
 * Skeleton — placeholder shimmer.
 *
 * Convenção: usar pra estados de loading em componentes que dependem de
 * dados do backend. SEMPRE medir largura/altura com classe explícita (ex:
 * `h-4 w-32`) — sem skeleton sem dimensão definida.
 */
import { cn } from '@/lib/utils'

export function Skeleton({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'skeleton rounded',
        className,
      )}
      {...props}
    />
  )
}

export function SkeletonText({
  lines = 3,
  className,
}: {
  lines?: number
  className?: string
}) {
  return (
    <div className={cn('space-y-2', className)}>
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton
          key={i}
          className={cn(
            'h-3',
            i === lines - 1 ? 'w-2/3' : 'w-full',
          )}
        />
      ))}
    </div>
  )
}