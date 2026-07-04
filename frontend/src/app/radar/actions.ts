'use server'

/**
 * Server actions para /radar.
 *
 * Server action resolveAlert delega ao backend via apiFetch (injeta JWT).
 */
import { cookies } from 'next/headers'
import { revalidatePath } from 'next/cache'

const API_URL = process.env.RADIANT_API_URL || 'http://localhost:8080'

export async function resolveAlert(id: number): Promise<void> {
  const token = (await cookies()).get('rn_jwt')?.value
  if (!token) throw new Error('Sem sessão')

  const res = await fetch(`${API_URL}/v1/radar/alerts/${id}/resolve`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
  })
  if (!res.ok) {
    throw new Error(`Falha ao resolver alerta: ${res.status}`)
  }
  revalidatePath('/radar')
  revalidatePath('/')
}