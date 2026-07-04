// API route /api/radar/alerts/[id]/resolve
//
// Validação 29 (C1 fix): bridge entre client component AlertCard e
// backend Go. Client faz POST /api/... → injetamos JWT do cookie →
// chamamos backend /v1/radar/alerts/:id/resolve.

import { NextRequest, NextResponse } from 'next/server'
import { cookies } from 'next/headers'

const API_URL = process.env.RADIANT_API_URL || 'http://localhost:8080'

export async function POST(
  _req: NextRequest,
  { params }: { params: { id: string } },
) {
  const id = Number(params.id)
  if (!Number.isFinite(id) || id <= 0) {
    return NextResponse.json({ error: 'id inválido' }, { status: 400 })
  }

  const token = (await cookies()).get('rn_jwt')?.value
  if (!token) {
    return NextResponse.json({ error: 'não autenticado' }, { status: 401 })
  }

  const r = await fetch(`${API_URL}/v1/radar/alerts/${id}/resolve`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
  })

  if (!r.ok) {
    const body = await r.text().catch(() => '')
    return NextResponse.json(
      { error: `backend ${r.status}`, detail: body.slice(0, 200) },
      { status: r.status },
    )
  }

  return NextResponse.json({ ok: true })
}