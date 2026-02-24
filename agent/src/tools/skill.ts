import { tool } from 'ai'
import { z } from 'zod'

interface SkillToolParams {
  useSkill: (skill: string) => Promise<{ content: string; description: string } | null>
}

export const getSkillTools = ({ useSkill }: SkillToolParams) => {
  const useSkillTool = tool({
    description: 'Use a skill if you think it is relevant to the current task. Returns the full skill content so you can follow its instructions.',
    inputSchema: z.object({
      skillName: z.string().describe('The name of the skill to use'),
      reason: z.string().describe('The reason why you think this skill is relevant to the current task'),
    }),
    execute: async ({ skillName, reason }) => {
      const skill = await useSkill(skillName)
      if (!skill) {
        return { success: false, skillName, reason, error: 'Skill not found' }
      }
      return {
        success: true,
        skillName,
        reason,
        description: skill.description,
        instructions: skill.content,
      }
    },
  })

  return {
    'use_skill': useSkillTool,
  }
}