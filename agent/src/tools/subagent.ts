import { tool } from 'ai'
import { z } from 'zod'
import { createAgent } from '../agent'
import { ModelConfig, AgentAuthContext, MCPConnection } from '../types'
import { AuthFetcher } from '..'
import { AgentAction, IdentityContext } from '../types/agent'
import { SubagentRegistry } from '../registry'

export interface SubagentToolParams {
  fetch: AuthFetcher
  model: ModelConfig
  identity: IdentityContext
  auth: AgentAuthContext
  mcpConnections?: MCPConnection[]
  registry: SubagentRegistry
  parentRunId?: string
  spawnDepth?: number
}

export const getSubagentTools = ({
  fetch,
  model,
  identity,
  auth,
  mcpConnections = [],
  registry,
  parentRunId,
  spawnDepth = 0,
}: SubagentToolParams) => {
  const botId = identity.botId.trim()
  const base = `/bots/${botId}/subagents`

  // ── CRUD tools (unchanged) ──────────────────────────────────────────

  const listSubagents = tool({
    description: 'List all registered sub-agent definitions for the current bot.',
    inputSchema: z.object({}),
    execute: async () => {
      if (!botId) throw new Error('bot_id is required')
      const response = await fetch(base, { method: 'GET' })
      return response.json()
    },
  })

  const createSubagent = tool({
    description: 'Create a new sub-agent definition (does not start it).',
    inputSchema: z.object({
      name: z.string().describe('Unique name for the sub-agent'),
      description: z.string().describe('What this sub-agent does'),
    }),
    execute: async ({ name, description }) => {
      if (!botId) throw new Error('bot_id is required')
      const response = await fetch(base, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, description }),
      })
      return response.json()
    },
  })

  const deleteSubagent = tool({
    description: 'Delete a sub-agent definition by ID.',
    inputSchema: z.object({
      id: z.string().describe('Sub-agent ID'),
    }),
    execute: async ({ id }) => {
      if (!botId) throw new Error('bot_id is required')
      const response = await fetch(`${base}/${id}`, { method: 'DELETE' })
      return response.status === 204 ? { success: true } : response.json()
    },
  })

  // ── Synchronous query (kept for simple inline Q&A) ──────────────────

  const querySubagent = tool({
    description:
      'Send a prompt to a sub-agent and wait for the result synchronously. ' +
      'Use this for quick questions; for long-running tasks use spawn_subagent instead.',
    inputSchema: z.object({
      name: z.string().describe('Name of the sub-agent to query'),
      query: z.string().describe('The prompt / task to send'),
    }),
    execute: async ({ name, query }) => {
      if (!botId) throw new Error('bot_id is required')
      const { target, contextMessages } = await resolveSubagent(name)
      const { askAsSubagent } = createSubagentAgent()
      const result = await askAsSubagent({
        messages: contextMessages,
        input: query,
        name: target.name,
        description: target.description,
      })
      const updatedMessages = [...contextMessages, ...result.messages]
      await saveContext(target.id, updatedMessages)
      return {
        success: true,
        result: result.messages[result.messages.length - 1].content,
      }
    },
  })

  // ── Async spawn / check / kill / steer ──────────────────────────────

  const spawnSubagent = tool({
    description:
      'Launch a sub-agent task in the background (fire-and-forget). ' +
      'Returns a runId you can use with check_subagent_run / kill_subagent_run / steer_subagent.',
    inputSchema: z.object({
      name: z.string().describe('Name of the sub-agent to spawn'),
      task: z.string().describe('The task / prompt to execute'),
    }),
    execute: async ({ name, task }) => {
      if (!botId) throw new Error('bot_id is required')
      const { target, contextMessages } = await resolveSubagent(name)
      const runId = registry.generateRunId()
      const abortController = new AbortController()

      const run = {
        runId,
        name: target.name,
        task,
        status: 'running' as const,
        abortController,
        spawnDepth: spawnDepth + 1,
        parentRunId,
        startedAt: Date.now(),
      }

      registry.register(run)

      // Fire-and-forget
      ;(async () => {
        try {
          const { askAsSubagent } = createSubagentAgent()
          const result = await askAsSubagent({
            messages: contextMessages,
            input: task,
            name: target.name,
            description: target.description,
            abortSignal: abortController.signal,
          })
          const updatedMessages = [...contextMessages, ...result.messages]
          try {
            await saveContext(target.id, updatedMessages)
          } catch (saveErr) {
            console.error(`[subagent:${runId}] failed to save context`, saveErr)
          }
          const lastMessage = result.messages[result.messages.length - 1]
          const lastContent = lastMessage?.content
          const summary = typeof lastContent === 'string'
            ? lastContent
            : lastContent != null ? JSON.stringify(lastContent) : '(no output)'
          registry.complete(runId, summary)
        } catch (err: unknown) {
          if (abortController.signal.aborted) return
          const message = err instanceof Error ? err.message : String(err)
          registry.fail(runId, message)
        }
      })()

      return {
        runId,
        name: target.name,
        status: 'running',
        message: `Sub-agent "${target.name}" spawned. Use check_subagent_run with runId "${runId}" to poll for results.`,
      }
    },
  })

  const checkSubagentRun = tool({
    description: 'Check the status and result of a spawned sub-agent run.',
    inputSchema: z.object({
      run_id: z.string().describe('The runId returned by spawn_subagent'),
    }),
    execute: async ({ run_id }) => {
      const run = registry.get(run_id)
      if (!run) return { error: `Run not found: ${run_id}` }
      return {
        runId: run.runId,
        name: run.name,
        task: run.task,
        status: run.status,
        result: run.result ?? null,
        error: run.error ?? null,
        elapsed_ms: (run.endedAt ?? Date.now()) - run.startedAt,
      }
    },
  })

  const killSubagentRun = tool({
    description:
      'Abort a running sub-agent. Accepts either a runId or a sub-agent name (kills the first active run matching that name).',
    inputSchema: z.object({
      run_id: z.string().optional().describe('The runId to kill'),
      name: z.string().optional().describe('Sub-agent name to kill (first active run)'),
    }),
    execute: async ({ run_id, name }) => {
      let target: string | undefined = run_id
      if (!target && name) {
        const run = registry.findByName(name)
        if (!run) return { error: `No active run found for sub-agent: ${name}` }
        target = run.runId
      }
      if (!target) return { error: 'Provide either run_id or name' }
      const killed = registry.abort(target)
      return {
        success: killed > 0,
        killed_count: killed,
        message: killed > 0 ? `Aborted ${killed} run(s)` : 'Run was not active',
      }
    },
  })

  const steerSubagent = tool({
    description:
      'Redirect a running sub-agent: aborts the current run and spawns a new one ' +
      'with the same conversation context plus a new instruction. ' +
      'Returns the new runId.',
    inputSchema: z.object({
      name: z.string().describe('Name of the sub-agent to steer'),
      new_task: z.string().describe('New instruction to give the sub-agent'),
      run_id: z.string().optional().describe('Specific runId to steer (optional; defaults to first active run by name)'),
    }),
    execute: async ({ name, new_task, run_id }) => {
      if (!botId) throw new Error('bot_id is required')

      // Kill existing run
      const existingRun = run_id ? registry.get(run_id) : registry.findByName(name)
      if (existingRun && existingRun.status === 'running') {
        registry.abort(existingRun.runId)
      }

      // Re-spawn with new task
      const { target, contextMessages } = await resolveSubagent(name)
      const newRunId = registry.generateRunId()
      const abortController = new AbortController()

      const run = {
        runId: newRunId,
        name: target.name,
        task: new_task,
        status: 'running' as const,
        abortController,
        spawnDepth: spawnDepth + 1,
        parentRunId,
        startedAt: Date.now(),
      }
      registry.register(run)

      ;(async () => {
        try {
          const { askAsSubagent } = createSubagentAgent()
          const result = await askAsSubagent({
            messages: contextMessages,
            input: new_task,
            name: target.name,
            description: target.description,
            abortSignal: abortController.signal,
          })
          const updatedMessages = [...contextMessages, ...result.messages]
          try {
            await saveContext(target.id, updatedMessages)
          } catch (saveErr) {
            console.error(`[subagent:${newRunId}] failed to save context`, saveErr)
          }
          const lastMessage = result.messages[result.messages.length - 1]
          const lastContent = lastMessage?.content
          const summary = typeof lastContent === 'string'
            ? lastContent
            : lastContent != null ? JSON.stringify(lastContent) : '(no output)'
          registry.complete(newRunId, summary)
        } catch (err: unknown) {
          if (abortController.signal.aborted) return
          const message = err instanceof Error ? err.message : String(err)
          registry.fail(newRunId, message)
        }
      })()

      return {
        previous_run_id: existingRun?.runId ?? null,
        new_run_id: newRunId,
        name: target.name,
        status: 'running',
        message: `Sub-agent "${target.name}" steered with new task. Poll check_subagent_run("${newRunId}").`,
      }
    },
  })

  const listSubagentRuns = tool({
    description: 'List all active and recent sub-agent runs with their status.',
    inputSchema: z.object({
      active_only: z.boolean().optional().describe('Only show running tasks (default: false)'),
    }),
    execute: async ({ active_only }) => {
      registry.sweep()
      const runs = active_only ? registry.listActive() : registry.list()
      return {
        count: runs.length,
        runs: runs.map(({ abortController: _, ...rest }) => rest),
      }
    },
  })

  // ── Helpers ─────────────────────────────────────────────────────────

  async function resolveSubagent(name: string) {
    const listResponse = await fetch(base, { method: 'GET' })
    const listPayload = await listResponse.json()
    const items = Array.isArray(listPayload?.items) ? listPayload.items : []
    const target = items.find((item: { name?: string }) => item?.name === name)
    if (!target?.id) throw new Error(`Sub-agent not found: ${name}`)

    const contextResponse = await fetch(`${base}/${target.id}/context`, { method: 'GET' })
    const contextPayload = await contextResponse.json()
    const contextMessages = Array.isArray(contextPayload?.messages) ? contextPayload.messages : []
    return { target, contextMessages }
  }

  async function saveContext(subagentId: string, messages: unknown[]) {
    await fetch(`${base}/${subagentId}/context`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ messages }),
    })
  }

  function createSubagentAgent() {
    return createAgent(
      {
        model,
        allowedActions: [
          AgentAction.Web,
          AgentAction.Skill,
          AgentAction.Memory,
        ],
        mcpConnections,
        identity,
        auth,
      },
      fetch,
    )
  }

  return {
    list_subagents: listSubagents,
    create_subagent: createSubagent,
    delete_subagent: deleteSubagent,
    query_subagent: querySubagent,
    spawn_subagent: spawnSubagent,
    check_subagent_run: checkSubagentRun,
    kill_subagent_run: killSubagentRun,
    steer_subagent: steerSubagent,
    list_subagent_runs: listSubagentRuns,
  }
}
