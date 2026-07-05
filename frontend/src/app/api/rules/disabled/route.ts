// /api/rules/disabled — proxy que adiciona JWT do cookie pra listar
// regras desabilitadas por IF do backend.

import { NextRequest, NextResponse } from 'next/server'
import { cookies } from 'next/headers'

const API_URL = process.env.RADIANT_API_URL || 'http://localhost:8080'

export async function GET(_req: NextRequest) {
  const cookieStore = cookies()
  const token = cookieStore.get('rn_jwt')?.value

  const headers: Record<string, string> = {}
  if (token) headers.Authorization = `Bearer ${token}`

  const response = await fetch(`${API_URL}/v1/rules/disabled`, { headers })
  const body = await response.text()
  return new NextResponse(body, {
    status: response.status,
    headers: {
      'Content-Type': response.headers.get('Content-Type') || 'application/json',
    },
  })
}