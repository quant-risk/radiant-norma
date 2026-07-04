/**
 * Format utilities — pt-BR defaults.
 *
 * Convenção: Intl.* APIs (sem deps externas). Pra datas relativas,
 * date-fns já cobre tudo com locale ptBR.
 */
import { format, formatDistanceToNow, parseISO } from 'date-fns'
import { ptBR } from 'date-fns/locale'

/** "R$ 1.234,56" */
export function formatBRL(value: number): string {
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
    minimumFractionDigits: 2,
  }).format(value)
}

/** "1.234.567" (sem decimais) */
export function formatNumber(value: number): string {
  return new Intl.NumberFormat('pt-BR').format(value)
}

/** "45,3%" */
export function formatPercent(value: number, digits = 1): string {
  return new Intl.NumberFormat('pt-BR', {
    style: 'percent',
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(value / 100)
}

/** "05/07/2026" */
export function formatDate(iso: string | Date): string {
  const d = typeof iso === 'string' ? parseISO(iso) : iso
  return format(d, 'dd/MM/yyyy', { locale: ptBR })
}

/** "05/07/2026 14:32" */
export function formatDateTime(iso: string | Date): string {
  const d = typeof iso === 'string' ? parseISO(iso) : iso
  return format(d, "dd/MM/yyyy HH:mm", { locale: ptBR })
}

/** "há 5 minutos" */
export function formatRelative(iso: string | Date): string {
  const d = typeof iso === 'string' ? parseISO(iso) : iso
  return formatDistanceToNow(d, {
    addSuffix: true,
    locale: ptBR,
  })
}

/** "5 min atrás" (compacto, sem prepositions) */
export function formatRelativeCompact(iso: string | Date): string {
  const d = typeof iso === 'string' ? parseISO(iso) : iso
  const seconds = Math.floor((Date.now() - d.getTime()) / 1000)
  if (seconds < 60) return 'agora'
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes} min atrás`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h atrás`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d atrás`
  const months = Math.floor(days / 30)
  if (months < 12) return `${months}m atrás`
  return `${Math.floor(months / 12)}a atrás`
}