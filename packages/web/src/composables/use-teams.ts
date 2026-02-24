import { ref, computed } from 'vue'
import { listTeams, listTeamMembers, type Team } from '../lib/api-teams'

/**
 * Composable for loading and managing team data.
 * Fetches teams and enriches each team with its member bot IDs.
 */
export function useTeams() {
  const teams = ref<Team[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function refreshTeams() {
    loading.value = true
    error.value = null
    try {
      const rawTeams = await listTeams()
      // Enrich each team with member IDs in parallel.
      await Promise.all(rawTeams.map(async (team) => {
        try {
          const members = await listTeamMembers(team.id)
          team.memberBotIds = members.map(m => m.target_bot_id)
        } catch {
          team.memberBotIds = []
        }
      }))
      teams.value = rawTeams
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load teams'
    } finally {
      loading.value = false
    }
  }

  refreshTeams()

  /** Set of all bot IDs that belong to any team (manager + members). */
  const teamBotIds = computed(() => {
    const ids = new Set<string>()
    for (const team of teams.value) {
      if (team.manager_bot_id) ids.add(team.manager_bot_id)
      for (const mid of team.memberBotIds ?? []) ids.add(mid)
    }
    return ids
  })

  return { teams, loading, error, teamBotIds, refreshTeams }
}
