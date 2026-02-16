import { client } from '@memoh/sdk/client'
import router from '@/router'

/**
 * Configure the SDK client with base URL, auth interceptor, and 401 handling.
 * Call this once at app startup (main.ts).
 *
 * NOTE: Prefer using auto-generated SDK functions (e.g. `getBots()`, `postProviders()`)
 * from `@memoh/sdk` over direct `client.get()`/`client.post()` calls.
 * SDK functions provide full type safety and stay in sync with the API schema.
 */
export function setupApiClient() {
  const apiBaseUrl = import.meta.env.VITE_API_URL?.trim() || '/api'

  client.setConfig({ baseUrl: apiBaseUrl })

  // Add auth token to every request
  client.interceptors.request.use((request) => {
    const token = localStorage.getItem('token')
    if (token) {
      request.headers.set('Authorization', `Bearer ${token}`)
    }
    return request
  })

  // Handle 401 responses globally
  client.interceptors.response.use((response) => {
    if (response.status === 401) {
      localStorage.removeItem('token')
      router.replace({ name: 'Login' })
    }
    return response
  })
}
