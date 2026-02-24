import { AuthFetcher } from '..'
import { AgentAction, AgentAuthContext, IdentityContext, MCPConnection, ModelConfig } from '../types'
import { ToolSet } from 'ai'
import { getWebTools } from './web'
import { getSubagentTools } from './subagent'
import { getSkillTools } from './skill'
import { getCallAgentTools } from './call-agent'
import { SubagentRegistry, getGlobalRegistry } from '../registry'

export interface ToolsParams {
  fetch: AuthFetcher
  model: ModelConfig
  backgroundModel?: ModelConfig
  identity: IdentityContext
  auth: AgentAuthContext
  enableSkill: (skill: string) => Promise<{ content: string; description: string } | null>
  mcpConnections?: MCPConnection[]
  registry?: SubagentRegistry
  parentRunId?: string
  spawnDepth?: number
  teamMembers?: string[]
  callDepth?: number
}

export const getTools = (
  actions: AgentAction[],
  { fetch, model, backgroundModel, identity, auth, enableSkill, mcpConnections = [], registry, parentRunId, spawnDepth = 0, teamMembers = [], callDepth = 0 }: ToolsParams
) => {
  const tools: ToolSet = {}
  if (actions.includes(AgentAction.Web)) {
    const webTools = getWebTools()
    Object.assign(tools, webTools)
  }
  if (actions.includes(AgentAction.Subagent)) {
    const subagentTools = getSubagentTools({
      fetch,
      model,
      backgroundModel,
      identity,
      auth,
      mcpConnections,
      registry: registry ?? getGlobalRegistry(),
      parentRunId,
      spawnDepth,
    })
    Object.assign(tools, subagentTools)
  }
  if (actions.includes(AgentAction.Skill)) {
    const skillTools = getSkillTools({ useSkill: enableSkill })
    Object.assign(tools, skillTools)
  }
  if (teamMembers.length > 0) {
    const callAgentTools = getCallAgentTools({ fetch, identity, auth, allowedBotIds: teamMembers, callDepth })
    Object.assign(tools, callAgentTools)
  }
  return tools
}
