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

    const apiUrl = process.env.RADIANT_API_URL || 'http://localhost:8080'

    const res = await fetch(`${apiUrl}/v1/sta/submit`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
        'X-IF-ID': 'demo',
      },
      body: JSON.stringify(body),
    })

    const data = await res.json()

    if (!res.ok) {
      return NextResponse.json({ error: data.error || 'Submission failed' }, { status: res.status })
    }

    return NextResponse.json(data)
  } catch (err) {
    console.error('[STA submit]', err)
    return NextResponse.json({ error: 'Server error' }, { status: 500 })
  }
}
