import { Elysia } from 'elysia'
import { OpenRouter } from '@openrouter/sdk'
import z from 'zod'

const ImageGenerateBody = z.object({
  apiKey: z.string(),
  model: z.string(),
  prompt: z.string(),
})

export const imageModule = new Elysia({ prefix: '/image' })
  .post('/generate', async ({ body }) => {
    const client = new OpenRouter({ apiKey: body.apiKey })

    const response = await client.chat.send({
      chatGenerationParams: {
        model: body.model,
        messages: [{ role: 'user' as const, content: body.prompt }],
        modalities: ['text', 'image'],
      },
    })

    const message = response.choices?.[0]?.message
    if (!message) {
      return { success: false, error: 'No response from model' }
    }

    if (message.images && message.images.length > 0) {
      for (const img of message.images) {
        const url = img.imageUrl?.url
        if (typeof url === 'string' && url.startsWith('data:image/')) {
          const commaIdx = url.indexOf(',')
          if (commaIdx > 0) {
            const base64 = url.substring(commaIdx + 1)
            const mimeMatch = url.match(/^data:(image\/[^;]+);/)
            const mimeType = mimeMatch ? mimeMatch[1] : 'image/png'
            return { success: true, image: base64, mimeType }
          }
        }
        if (typeof url === 'string' && (url.startsWith('http://') || url.startsWith('https://'))) {
          const imgResp = await fetch(url)
          if (!imgResp.ok) continue
          const buf = await imgResp.arrayBuffer()
          const base64 = Buffer.from(buf).toString('base64')
          const ct = imgResp.headers.get('content-type') || 'image/png'
          return { success: true, image: base64, mimeType: ct }
        }
      }
      return {
        success: false,
        error: `images field had ${message.images.length} entries but none contained decodable image data`,
      }
    }

    const content = message.content
    if (typeof content === 'string' && content.startsWith('data:image/')) {
      const commaIdx = content.indexOf(',')
      if (commaIdx > 0) {
        const base64 = content.substring(commaIdx + 1)
        return { success: true, image: base64, mimeType: 'image/png' }
      }
    }

    const preview = typeof content === 'string' ? content.slice(0, 200) : JSON.stringify(content)?.slice(0, 200)
    return {
      success: false,
      error: `No image in response (model: ${body.model}, content_preview: ${preview})`,
    }
  }, {
    body: ImageGenerateBody,
  })
