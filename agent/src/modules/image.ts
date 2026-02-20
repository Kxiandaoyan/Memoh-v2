import { Elysia } from 'elysia'

const OPENROUTER_BASE = 'https://openrouter.ai/api/v1'

interface ORImageUrl {
  url: string
}

interface ORImage {
  image_url?: ORImageUrl
}

interface ORContentPart {
  type?: string
  image_url?: ORImageUrl
  b64_json?: string
  data?: string
}

interface ORMessage {
  role: string
  content?: string | ORContentPart[] | null
  images?: ORImage[]
}

interface ORChoice {
  message?: ORMessage
}

interface ORChatResponse {
  choices?: ORChoice[]
  error?: { message?: string }
}

async function resolveImageUrl(url: string): Promise<{ base64: string; mimeType: string } | null> {
  if (url.startsWith('data:image/')) {
    const comma = url.indexOf(',')
    if (comma > 0) {
      const mimeMatch = url.match(/^data:(image\/[^;]+);/)
      return { base64: url.substring(comma + 1), mimeType: mimeMatch?.[1] ?? 'image/png' }
    }
    return null
  }

  if (url.startsWith('http://') || url.startsWith('https://')) {
    const resp = await fetch(url)
    if (!resp.ok) return null
    const buf = await resp.arrayBuffer()
    if (buf.byteLength < 100) return null
    const ct = resp.headers.get('content-type') || 'image/png'
    return { base64: Buffer.from(buf).toString('base64'), mimeType: ct.split(';')[0] }
  }

  if (url.length > 200) {
    return { base64: url, mimeType: 'image/png' }
  }

  return null
}

async function extractImage(resp: ORChatResponse, model: string): Promise<{ base64: string; mimeType: string } | string> {
  if (resp.error?.message) {
    return `API error: ${resp.error.message}`
  }

  const msg = resp.choices?.[0]?.message
  if (!msg) return 'No response from model'

  if (msg.images && msg.images.length > 0) {
    for (const img of msg.images) {
      const url = img.image_url?.url
      if (url) {
        const resolved = await resolveImageUrl(url)
        if (resolved) return resolved
      }
    }
    return `images field had ${msg.images.length} entries but none contained decodable data`
  }

  const content = msg.content
  if (Array.isArray(content)) {
    for (const part of content) {
      if (part.image_url?.url) {
        const resolved = await resolveImageUrl(part.image_url.url)
        if (resolved) return resolved
      }
      if (part.b64_json && part.b64_json.length > 100) {
        return { base64: part.b64_json, mimeType: 'image/png' }
      }
    }
    return `multimodal response had ${content.length} parts but none contained image data`
  }

  if (typeof content === 'string') {
    const resolved = await resolveImageUrl(content)
    if (resolved) return resolved
  }

  const preview = typeof content === 'string' ? content.slice(0, 200) : JSON.stringify(content)?.slice(0, 200)
  return `No image in response (model: ${model}, content_preview: ${preview})`
}

export const imageModule = new Elysia({ prefix: '/image' })
  .post('/generate', async ({ request }) => {
    try {
      const raw = await request.json() as Record<string, unknown>
      const apiKey = typeof raw?.apiKey === 'string' ? raw.apiKey : ''
      const model = typeof raw?.model === 'string' ? raw.model : ''
      const prompt = typeof raw?.prompt === 'string' ? raw.prompt : ''

      if (!apiKey || !model || !prompt) {
        return { success: false, error: 'Missing required fields: apiKey, model, prompt' }
      }

      const resp = await fetch(`${OPENROUTER_BASE}/chat/completions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${apiKey}`,
        },
        body: JSON.stringify({
          model,
          messages: [{ role: 'user', content: prompt }],
          modalities: ['image', 'text'],
        }),
      })

      if (!resp.ok) {
        const text = await resp.text()
        return { success: false, error: `OpenRouter API error (${resp.status}): ${text.slice(0, 500)}` }
      }

      const data = await resp.json() as ORChatResponse

      const result = await extractImage(data, model)
      if (typeof result === 'string') {
        return { success: false, error: result }
      }
      return { success: true, image: result.base64, mimeType: result.mimeType }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      console.error('imagegen endpoint error:', msg)
      return { success: false, error: msg }
    }
  })
