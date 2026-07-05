// /v1-api/events/stream — proxy SSE que adiciona JWT do cookie.
//
// Browser EventSource nativo não consegue setar headers custom (Authorization)
// por design. Solução: Next.js Route Handler que:
//   1. Lê cookie httpOnly rn_jwt (server-side, next/headers)
//   2. Abre stream com backend Go (com Authorization header)
//   3. Pipes response.body do backend pro browser (sem buffer)
//
// Runtime 'edge' é necessário pra streaming real (Node runtime tem buffer
// chunked de 4KB que adiciona latência perceptível).

import { cookies } from 'next/headers'

export const runtime = 'edge'
export const dynamic = 'force-dynamic'

const API_URL = process.env.RADIANT_API_URL || 'http://localhost:8080'

export async function GET(req: Request) {
  const cookieStore = cookies()
  const token = cookieStore.get('rn_jwt')?.value

  const headers: Record<string, string> = {
    Accept: 'text/event-stream',
  }
  if (token) headers.Authorization = `Bearer ${token}`

  // X-IF-ID fallback em dev (dev mode backend aceita)
  const url = new URL(req.url)
  const ifID = url.searchParams.get('if_id')
  if (ifID) headers['X-IF-ID'] = ifID

  const backendResp = await fetch(`${API_URL}/v1/events/stream`, {
    headers,
    // Importante: NÃO setar Cache-Control no request
  })

  if (!backendResp.ok || !backendResp.body) {
    return new Response(
      `event: error\ndata: {"status":${backendResp.status}}\n\n`,
      {
        status: backendResp.status,
        headers: {
          'Content-Type': 'text/event-stream',
          'Cache-Control': 'no-cache',
        },
      },
    )
  }

  // Pass-through do stream do backend. Headers corretos pra SSE no browser.
  return new Response(backendResp.body, {
    status: 200,
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache, no-transform',
      Connection: 'keep-alive',
      'X-Accel-Buffering': 'no',
    },
  })
}