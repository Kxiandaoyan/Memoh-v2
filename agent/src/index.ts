import { Elysia } from 'elysia'
import { chatModule } from './modules/chat'
import { corsMiddleware } from './middlewares/cors'
import { errorMiddleware } from './middlewares/error'
import { loadConfig, getBaseUrl as getBaseUrlByConfig } from '@memoh/config'
import { normalizeBaseUrl } from './utils/url'

const config = loadConfig('../config.toml')

const MAX_BUN_IDLE_TIMEOUT = 255

export const getBaseUrl = () => {
  return getBaseUrlByConfig(config)
}

export type AuthFetcher = (
  url: string,
  options?: RequestInit,
) => Promise<Response>;
export const createAuthFetcher = (bearer: string | undefined): AuthFetcher => {
  return async (url: string, options?: RequestInit) => {
    const requestOptions = options ?? {}
    const headers = new Headers(requestOptions.headers || {})
    if (bearer && !headers.has('Authorization')) {
      headers.set('Authorization', `Bearer ${bearer}`)
    }

    const baseURL = getBaseUrl()
    const requestURL = /^https?:\/\//i.test(url)
      ? url
      : new URL(url, `${normalizeBaseUrl(baseURL)}/`).toString()

    return await fetch(requestURL, {
      ...requestOptions,
      headers,
    })
  }
}

const app = new Elysia()
  .use(corsMiddleware)
  .use(errorMiddleware)
  .get('/health', () => ({
    status: 'ok',
  }))
  .use(chatModule)
  .listen({
    port: config.agent_gateway.port ?? 8081,
    hostname: config.agent_gateway.host ?? '127.0.0.1',
    idleTimeout: MAX_BUN_IDLE_TIMEOUT,
  })

console.log(
  `Agent Gateway is running at ${app.server?.hostname}:${app.server?.port}`,
)
