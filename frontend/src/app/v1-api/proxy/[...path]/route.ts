// /proxy route — server-side forward para backend Go.
//
// Páginas Server Component não podem setar cookies (mas podem ler).
// Para mutações client-side (ex: POST /v1/radar/alerts/{id}/resolve),
// proxy route adiciona JWT do cookie httpOnly e forwards.

import { NextRequest, NextResponse } from 'next/server'
import { cookies } from 'next/headers'

const API_URL = process.env.RADIANT_API_URL || 'http://localhost:8080'

export async function POST(
  req: NextRequest,
  { params }: { params: { path: string[] } },
) {
  const path = '/' + (params.path?.join('/') ?? '')
  const cookieStore = cookies()
  const token = cookieStore.get('rn_jwt')?.value

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (token) headers.Authorization = `Bearer ${token}`

  const body = await req.text().catch(() => '')
  const response = await fetch(`${API_URL}${path}`, {
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

export async function GET(
  req: NextRequest,
  { params }: { params: { path: string[] } },
) {
  const path = '/' + (params.path?.join('/') ?? '')
  const cookieStore = cookies()
  const token = cookieStore.get('rn_jwt')?.value

  const headers: Record<string, string> = {}
  if (token) headers.Authorization = `Bearer ${token}`

  const url = `${API_URL}${path}${req.nextUrl.search || ''}`
  const response = await fetch(url, { method: 'GET', headers })
  const responseBody = await response.text()
  return new NextResponse(responseBody, {
    status: response.status,
    headers: {
      'Content-Type': response.headers.get('Content-Type') || 'application/json',
    },
  })
}
