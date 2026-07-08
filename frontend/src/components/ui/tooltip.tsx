'use client'

/**
 * Tooltip — implementação leve com glass background.
 */
import * as React from 'react'
import { cn } from '@/lib/utils'

export interface TooltipProps {
  content: React.ReactNode
  children: React.ReactElement
  side?: 'top' | 'bottom' | 'left' | 'right'
  delay?: number
}

export function Tooltip({
  content,
  children,
  side = 'top',
  delay = 240,
}: TooltipProps) {
  const [open, setOpen] = React.useState(false)
  const timeoutRef = React.useRef<NodeJS.Timeout>()

  const show = () => {
    timeoutRef.current = setTimeout(() => setOpen(true), delay)
  }
  const hide = () => {
    if (timeoutRef.current) clearTimeout(timeoutRef.current)
    setOpen(false)
  }

  const positions = {
    top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
    bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
    left: 'right-full top-1/2 -translate-y-1/2 mr-2',
    right: 'left-full top-1/2 -translate-y-1/2 ml-2',
  }

  return (
    <span
      className="relative inline-flex"
      onMouseEnter={show}
      onMouseLeave={hide}
      onFocus={show}
      onBlur={hide}
    >
      {children}
      {open && (
        <span
          role="tooltip"
          className={cn(
            'absolute z-50 px-2.5 py-1.5 rounded-md text-2xs font-medium tracking-tight',
            'bg-ink text-ink-inverse shadow-lg whitespace-nowrap pointer-events-none',
            'animate-fade-in-fast',
            positions[side],
          )}
        >
          {content}
        </span>
      )}
    </span>
  )
}