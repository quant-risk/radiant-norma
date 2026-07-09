// API route: POST /api/generate/xml
// Proxies to backend /v1/generate/{cadoc} to generate XML from CanonicalDocument.
// Reads JWT cookie server-side and forwards Authorization header.
import { NextRequest, NextResponse } from 'next/server'
import { cookies } from 'next/headers'

export async function POST(req: NextRequest) {
  try {
    const body = await req.json()
    const cookieStore = cookies()
    const token = cookieStore.get('rn_jwt')?.value

    if (!token) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    let ifId = 'demo'
    if (token.startsWith('dev:')) {
      const parts = token.split(':')
      ifId = parts[1] || 'demo'
    }

    const cadoc = body.cadoc_code || body.cadoc
    if (!cadoc) {
      return NextResponse.json({ error: 'cadoc_code is required' }, { status: 400 })
    }

    const apiUrl = process.env.RADIANT_API_URL || 'http://localhost:8080'

    const res = await fetch(`${apiUrl}/v1/generate/${cadoc}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
        'X-IF-ID': ifId,
      },
      body: JSON.stringify(body),
    })

    const data = await res.json()

    if (!res.ok) {
      return NextResponse.json(data, { status: res.status })
    }

    return NextResponse.json(data)
  } catch (err) {
    console.error('[generate/xml]', err)
    return NextResponse.json({ error: 'Server error' }, { status: 500 })
  }
}
