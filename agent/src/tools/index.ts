import { AuthFetcher } from '..'
import { AgentAction, AgentAuthContext, IdentityContext, MCPConnection, ModelConfig } from '../types'
import { ToolSet } from 'ai'
import { getWebTools } from './web'
import { getSubagentTools } from './subagent'
import { getSkillTools } from './skill'
import { SubagentRegistry, getGlobalRegistry } from '../registry'

export interface ToolsParams {
  fetch: AuthFetcher
  model: ModelConfig
  backgroundModel?: ModelConfig
  identity: IdentityContext
  auth: AgentAuthContext
  enableSkill: (skill: string) => void
  mcpConnections?: MCPConnection[]
  registry?: SubagentRegistry
  parentRunId?: string
  spawnDepth?: number
}

export const getTools = (
  actions: AgentAction[],
  { fetch, model, backgroundModel, identity, auth, enableSkill, mcpConnections = [], registry, parentRunId, spawnDepth = 0 }: ToolsParams
) => {
  const tools: ToolSet = {}
  if (actions.includes(AgentAction.Web)) {
    const webTools = getWebTools()
    Object.assign(tools, webTools)
  }
  if (actions.includes(AgentAction.Subagent)) {
    const subagentTools = getSubagentTools({
      fetch,
      model: backgroundModel ?? model,
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
  return tools
}
