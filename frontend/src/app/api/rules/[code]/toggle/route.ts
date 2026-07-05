// /api/rules/[code]/toggle — proxy que adiciona JWT do cookie.
//
// Server-side forward que injeta Authorization header antes de chamar
// backend Go /v1/rules/{code}/toggle.
//
// Body é repassa ao backend (expected_state pra optimistic concurrency).
//
// Sprint 12 (v3.5.0) — C32.19: valida formato de rule_code antes de
// chamar backend. Bloqueia typos extremos e payloads maliciosos na
// borda (defense in depth — backend também valida).

import { NextRequest, NextResponse } from 'next/server'
import { cookies } from 'next/headers'

const API_URL = process.env.RADIANT_API_URL || 'http://localhost:8080'

// C32.19: rule_code válido = [A-Z][0-9]{1,3} (B12, F23, S05, C001).
// Mesmo regex do backend handler (defense in depth).
const VALID_RULE_CODE = /^[A-Z][0-9]{1,3}$/

export async function POST(
  req: NextRequest,
  { params }: { params: { code: string } },
) {
  const code = params.code

  // C32.19: valida formato antes de chamar backend
  if (!VALID_RULE_CODE.test(code)) {
    return NextResponse.json(
      { error: 'invalid rule code format (expected [A-Z][0-9]{1,3})' },
      { status: 400 },
    )
  }

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