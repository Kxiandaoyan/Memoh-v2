import { tool } from 'ai'
import { z } from 'zod'
import type { AuthFetcher } from '..'
import type { AgentAuthContext, IdentityContext } from '../types'

const MAX_CALL_DEPTH = 3
const DEFAULT_TIMEOUT_MS = 60_000

export interface CallAgentToolParams {
  fetch: AuthFetcher
  identity: IdentityContext
  auth: AgentAuthContext
  /** Bot IDs this agent is permitted to call. */
  allowedBotIds: string[]
  /** Current recursive call depth (propagated to target). */
  callDepth?: number
}

export const getCallAgentTools = ({
  fetch,
  identity,
  auth,
  allowedBotIds,
  callDepth = 0,
}: CallAgentToolParams) => {
  const allowedSet = new Set(allowedBotIds)

  const call_agent = tool({
    description:
      'Delegate a task to a team member bot by its bot_id. ' +
      'The target bot will process the message with its own memory, tools, and persona, ' +
      'then return its response. ' +
      'Use this for parallel or specialised work (e.g. researcher, writer, analyst). ' +
      'Only bots listed in your Team section are callable.',
    inputSchema: z.object({
      bot_id: z
        .string()
        .describe('The bot_id of the team member to call (must be in your Team section).'),
      message: z
        .string()
        .describe('The task or question to send to the target bot.'),
      wait_for_result: z
        .boolean()
        .optional()
        .default(true)
        .describe('If true (default), wait synchronously for the result. If false, fire-and-forget.'),
      timeout_seconds: z
        .number()
        .min(5)
        .max(120)
        .optional()
        .default(60)
        .describe('Maximum seconds to wait for the result (5–120, default 60).'),
    }),
    execute: async ({ bot_id, message, wait_for_result = true, timeout_seconds = 60 }) => {
      // Safety: reject if not in whitelist.
      if (!allowedSet.has(bot_id)) {
        return {
          success: false,
          error: `bot_id "${bot_id}" is not in your permitted team members list.`,
        }
      }

      // Safety: reject if call depth limit reached.
      if (callDepth >= MAX_CALL_DEPTH) {
        return {
          success: false,
          error: `call_agent depth limit (${MAX_CALL_DEPTH}) exceeded. Cannot make further nested calls.`,
        }
      }

      const callerBotId = identity.botId
      if (!callerBotId) {
        return { success: false, error: 'Could not determine caller bot_id from identity context.' }
      }

      const baseUrl = auth.baseUrl.replace(/\/$/, '')
      const url = `${baseUrl}/bots/${bot_id}/agent-call`

      const timeoutMs = Math.min(Math.max(timeout_seconds * 1000, 5_000), DEFAULT_TIMEOUT_MS * 2)

      try {
        const resp = await fetch(url, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${auth.bearer}`,
          },
          body: JSON.stringify({
            caller_bot_id: callerBotId,
            message,
            async: !wait_for_result,
            call_depth: callDepth + 1,
          }),
          signal: AbortSignal.timeout(timeoutMs),
        })

        if (resp.status === 429) {
          return { success: false, error: 'call_agent depth limit exceeded on server side.' }
        }
        if (resp.status === 403) {
          return { success: false, error: `Permission denied: cannot call bot "${bot_id}".` }
        }
        if (!resp.ok) {
          const text = await resp.text().catch(() => resp.statusText)
          return { success: false, error: `agent-call request failed (${resp.status}): ${text}` }
        }

        const data = (await resp.json()) as { result?: string; status?: string; log_id?: string }

        if (!wait_for_result) {
          return { success: true, status: 'accepted', log_id: data.log_id }
        }

        return {
          success: true,
          bot_id,
          result: data.result ?? '',
          status: data.status ?? 'completed',
        }
      } catch (err) {
        if (err instanceof Error && err.name === 'TimeoutError') {
          return { success: false, error: `call_agent timed out after ${timeout_seconds}s for bot "${bot_id}".` }
        }
        const message = err instanceof Error ? err.message : String(err)
        return { success: false, error: `call_agent failed: ${message}` }
      }
    },
  })

  return { call_agent }
}
