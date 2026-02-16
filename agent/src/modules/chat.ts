import { Elysia, sse } from 'elysia'
import { generateText } from 'ai'
import z from 'zod'
import { createAgent } from '../agent'
import { createAuthFetcher, getBaseUrl } from '../index'
import { createModel } from '../model'
import { ModelConfig, AgentParams } from '../types'
import { bearerMiddleware } from '../middlewares/bearer'
import { AgentSkillModel, AllowedActionModel, AttachmentModel, IdentityContextModel, MCPConnectionModel, ModelConfigModel, ScheduleModel } from '../models'
import { allActions } from '../types'

const AgentModel = z.object({
  model: ModelConfigModel,
  activeContextTime: z.number(),
  channels: z.array(z.string()),
  currentChannel: z.string(),
  allowedActions: z.array(AllowedActionModel).optional().default(allActions),
  messages: z.array(z.any()),
  usableSkills: z.array(AgentSkillModel).optional().default([]),
  skills: z.array(z.string()),
  identity: IdentityContextModel,
  attachments: z.array(AttachmentModel).optional().default([]),
  mcpConnections: z.array(MCPConnectionModel).optional().default([]),
  botIdentity: z.string().optional().default(''),
  botSoul: z.string().optional().default(''),
  botTask: z.string().optional().default(''),
  allowSelfEvolution: z.boolean().optional().default(true),
})

type AgentBody = z.infer<typeof AgentModel>

function buildAgentParams(body: AgentBody, bearer: string): AgentParams {
  return {
    model: body.model as ModelConfig,
    activeContextTime: body.activeContextTime,
    channels: body.channels,
    currentChannel: body.currentChannel,
    allowedActions: body.allowedActions,
    identity: body.identity,
    auth: {
      bearer,
      baseUrl: getBaseUrl(),
    },
    skills: body.usableSkills,
    mcpConnections: body.mcpConnections,
    botIdentity: body.botIdentity,
    botSoul: body.botSoul,
    botTask: body.botTask,
    allowSelfEvolution: body.allowSelfEvolution,
  }
}

export const chatModule = new Elysia({ prefix: '/chat' })
  .use(bearerMiddleware)
  .post('/', async ({ body, bearer }) => {
    console.log('chat', body)
    const authFetcher = createAuthFetcher(bearer)
    const { ask } = createAgent(buildAgentParams(body, bearer!), authFetcher)
    return ask({
      query: body.query,
      messages: body.messages,
      skills: body.skills,
      attachments: body.attachments,
    })
  }, {
    body: AgentModel.extend({
      query: z.string(),
    }),
  })
  .post('/stream', async function* ({ body, bearer }) {
    console.log('stream', body)
    try {
      const authFetcher = createAuthFetcher(bearer)
      const { stream } = createAgent(buildAgentParams(body, bearer!), authFetcher)
      for await (const action of stream({
        query: body.query,
        messages: body.messages,
        skills: body.skills,
        attachments: body.attachments,
      })) {
        yield sse(JSON.stringify(action))
      }
    } catch (error) {
      console.error(error)
      const message = error instanceof Error && error.message.trim()
        ? error.message
        : 'Internal server error'
      yield sse(JSON.stringify({
        type: 'error',
        message,
      }))
    }
  }, {
    body: AgentModel.extend({
      query: z.string(),
    }),
  })
  .post('/trigger-schedule', async ({ body, bearer }) => {
    console.log('trigger-schedule', body)
    const authFetcher = createAuthFetcher(bearer)
    const { triggerSchedule } = createAgent(buildAgentParams(body, bearer!), authFetcher)
    return triggerSchedule({
      schedule: body.schedule,
      messages: body.messages,
      skills: body.skills,
    })
  }, {
    body: AgentModel.extend({
      schedule: ScheduleModel,
    }),
  })
  .post('/summarize', async ({ body }) => {
    const model = createModel(body.model as ModelConfig)
    const { text } = await generateText({
      model,
      system: [
        'You are a precise conversation summarizer.',
        'Produce a concise summary of the conversation below.',
        'Preserve: key facts, user preferences, decisions made, action items, and important context.',
        'Omit: greetings, filler, tool call details, and redundant exchanges.',
        'Output ONLY the summary text, no preamble.',
      ].join('\n'),
      messages: body.messages,
    })
    return { summary: text }
  }, {
    body: z.object({
      model: ModelConfigModel,
      messages: z.array(z.any()),
    }),
  })
