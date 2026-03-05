import { HTTPMCPConnection, MCPConnection, SSEMCPConnection, StdioMCPConnection } from '../types'
import { createMCPClient } from '@ai-sdk/mcp'
import { AuthFetcher } from '../index'
import type { AgentAuthContext } from '../types/agent'
import { normalizeBaseUrl } from '../utils/url'

type MCPToolOptions = {
  botId?: string
  auth?: AgentAuthContext
  fetch?: AuthFetcher
}

export const getMCPTools = async (connections: MCPConnection[], options: MCPToolOptions = {}) => {
  const closeCallbacks: Array<() => Promise<void>> = []

  const getHTTPTools = async (connection: HTTPMCPConnection) => {
    const client = await createMCPClient({
      transport: {
        type: 'http',
        url: connection.url,
        headers: connection.headers,
      }
    })
    closeCallbacks.push(() => client.close())
    const tools = await client.tools()
    return tools
  }

  const getSSETools = async (connection: SSEMCPConnection) => {
    const client = await createMCPClient({
      transport: {
        type: 'sse',
        url: connection.url,
        headers: connection.headers,
      }
    })
    closeCallbacks.push(() => client.close())
    const tools = await client.tools()
    return tools
  }

  const getStdioTools = async (connection: StdioMCPConnection) => {
    if (!options.fetch || !options.botId || !options.auth) {
      throw new Error('stdio mcp requires auth fetcher and bot id')
    }
    const response = await options.fetch(`/bots/${options.botId}/mcp-stdio`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        name: connection.name,
        command: connection.command,
        args: connection.args ?? [],
        env: connection.env ?? {},
        cwd: connection.cwd ?? ''
      })
    })
    if (!response.ok) {
      const text = await response.text().catch(() => '')
      throw new Error(`mcp-stdio failed: ${response.status} ${text}`)
    }
    const data = await response.json().catch(() => ({} as { url?: string }))
    const rawUrl = typeof data?.url === 'string' ? data.url : ''
    if (!rawUrl) {
      throw new Error('mcp-stdio response missing url')
    }
    const baseUrl = options.auth.baseUrl ?? ''
    const url = rawUrl.startsWith('http')
      ? rawUrl
      : `${normalizeBaseUrl(baseUrl)}/${rawUrl.replace(/^\//, '')}`
    return await getHTTPTools({
      type: 'http',
      name: connection.name,
      url,
      headers: {
        'Authorization': `Bearer ${options.auth.bearer}`
      }
    })
  }

  const toolSets = await Promise.all(connections.map(async (connection) => {
    try {
      switch (connection.type) {
        case 'http':
          return getHTTPTools(connection)
        case 'sse':
          return getSSETools(connection)
        case 'stdio':
          return getStdioTools(connection)
        default:
          console.warn('unknown mcp connection type', connection)
          return {}
      }
    } catch (err) {
      console.warn(`[MCP] connection "${connection.name}" (${connection.type}) failed:`, (err as Error)?.message ?? err)
      return {}
    }
  }))

  const sanitizedTools = sanitizeToolsForGemini(Object.assign({}, ...toolSets))

  return {
    tools: sanitizedTools,
    close: async () => {
      await Promise.all(closeCallbacks.map(callback => callback()))
    }
  }
}

function sanitizeToolsForGemini(tools: Record<string, any>): Record<string, any> {
  const sanitized: Record<string, any> = {}

  for (const [name, tool] of Object.entries(tools)) {
    if (!tool || typeof tool !== 'object') {
      sanitized[name] = tool
      continue
    }

    sanitized[name] = {
      ...tool,
      parameters: sanitizeSchema(tool.parameters)
    }
  }

  return sanitized
}

function sanitizeSchema(schema: any): any {
  if (!schema || typeof schema !== 'object') return schema

  if (Array.isArray(schema)) {
    return schema.map(sanitizeSchema)
  }

  const result: any = {}

  for (const [key, value] of Object.entries(schema)) {
    if (key === 'enum' && Array.isArray(value)) {
      const filtered = value.filter(v => v !== '' && v != null)
      if (filtered.length > 0) {
        result[key] = filtered
      }
    } else if (value && typeof value === 'object') {
      result[key] = sanitizeSchema(value)
    } else {
      result[key] = value
    }
  }

  return result
}