import { tool } from 'ai'
import { z } from 'zod'
import { AgentAuthContext, IdentityContext } from '../types'
import { normalizeBaseUrl } from '../utils/url'

interface TierToolsOptions {
  auth: AgentAuthContext
  identity: IdentityContext
  fetch: (url: string, init?: RequestInit) => Promise<Response>
}

const enabledExtendedTools = new Set<string>()

export const getEnabledExtendedTools = () => [...enabledExtendedTools]

export const clearEnabledExtendedTools = () => enabledExtendedTools.clear()

export const createTierTools = ({ auth, identity, fetch: authFetch }: TierToolsOptions) => {
  const baseUrl = normalizeBaseUrl(auth.baseUrl)
  const botId = identity.botId.trim()

  const list_available_tools = tool({
    description: 'List extended tools that can be enabled on-demand to expand your capabilities.',
    inputSchema: z.object({}),
    execute: async () => {
      const url = `${baseUrl}/bots/${botId}/tools/extended`
      const res = await authFetch(url, {
        headers: { Authorization: `Bearer ${auth.bearer}` },
      })
      if (!res.ok) return { tools: [], error: `HTTP ${res.status}` }
      const data = await res.json().catch(() => ({ tools: [] }))
      return { tools: (data.tools ?? []).map((t: any) => ({ name: t.name, category: t.category })) }
    },
  })

  const enable_tools = tool({
    description: 'Enable extended tools for the current session. They will be available in subsequent messages.',
    inputSchema: z.object({
      tools: z.array(z.string()).describe('Tool names to enable'),
    }),
    execute: async ({ tools }) => {
      tools.forEach(t => enabledExtendedTools.add(t))
      return { enabled: tools, total: enabledExtendedTools.size }
    },
  })

  return { list_available_tools, enable_tools }
}
