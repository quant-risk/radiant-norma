// API route: POST /api/generate/wizard
// Proxies multipart file upload to backend /v1/generate/file/parse.
// Reads JWT cookie server-side and forwards Authorization header.
import { NextRequest, NextResponse } from 'next/server'
import { cookies } from 'next/headers'

export async function POST(req: NextRequest) {
  try {
    const cookieStore = cookies()
    const token = cookieStore.get('rn_jwt')?.value

    if (!token) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Extract if_id from token for X-IF-ID header.
    let ifId = 'demo'
    if (token.startsWith('dev:')) {
      const parts = token.split(':')
      ifId = parts[1] || 'demo'
    }

    const apiUrl = process.env.RADIANT_API_URL || 'http://localhost:8080'

    // Forward the multipart request to the backend.
    // We need to rebuild FormData because req.formData() can only be consumed once.
    const formData = await req.formData()
    const backendRes = await fetch(`${apiUrl}/v1/generate/file/parse`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'X-IF-ID': ifId,
      },
      body: formData,
    })

    const data = await backendRes.json()

    if (!backendRes.ok) {
      return NextResponse.json(data, { status: backendRes.status })
    }

    return NextResponse.json(data)
  } catch (err) {
    console.error('[generate/wizard]', err)
    return NextResponse.json({ error: 'Server error' }, { status: 500 })
  }
}
