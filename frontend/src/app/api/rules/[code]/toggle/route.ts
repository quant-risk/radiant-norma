// /api/rules/[code]/toggle — proxy que adiciona JWT do cookie.
//
// Server-side forward que injeta Authorization header antes de chamar
// backend Go /v1/rules/{code}/toggle.
//
// Body é repassa ao backend (expected_state pra optimistic concurrency).

import { NextRequest, NextResponse } from 'next/server'
import { cookies } from 'next/headers'

const API_URL = process.env.RADIANT_API_URL || 'http://localhost:8080'

export async function POST(
  req: NextRequest,
  { params }: { params: { code: string } },
) {
  const code = params.code
  const cookieStore = cookies()
  const token = cookieStore.get('rn_jwt')?.value

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (token) headers.Authorization = `Bearer ${token}`

  const body = await req.text().catch(() => '')

  const response = await fetch(`${API_URL}/v1/rules/${encodeURIComponent(code)}/toggle`, {
    method: 'POST',
    headers,
    body: body || undefined,
  })

  const responseBody = await response.text()
  return new NextResponse(responseBody, {
    status: response.status,
    headers: {
      'Content-Type': response.headers.get('Content-Type') || 'application/json',
    },
  })
}