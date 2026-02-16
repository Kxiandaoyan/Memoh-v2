/**
 * In-memory registry for tracking asynchronous sub-agent runs.
 * Each run is associated with an AbortController for cancellation.
 */

export type RunStatus = 'running' | 'completed' | 'aborted' | 'error'

export interface SubagentRun {
  runId: string
  name: string
  task: string
  status: RunStatus
  abortController: AbortController
  spawnDepth: number
  parentRunId?: string
  result?: string
  error?: string
  startedAt: number
  endedAt?: number
}

const DEFAULT_MAX_SPAWN_DEPTH = 1
const DEFAULT_MAX_CHILDREN = 5

export class SubagentRegistry {
  private runs = new Map<string, SubagentRun>()
  readonly maxSpawnDepth: number
  readonly maxChildren: number

  constructor(opts?: { maxSpawnDepth?: number; maxChildren?: number }) {
    this.maxSpawnDepth = opts?.maxSpawnDepth ?? DEFAULT_MAX_SPAWN_DEPTH
    this.maxChildren = opts?.maxChildren ?? DEFAULT_MAX_CHILDREN
  }

  /** Generate a unique run ID. */
  generateRunId(): string {
    return `run_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
  }

  /** Register a new run. Throws if limits are exceeded. */
  register(run: SubagentRun): void {
    const activeCount = this.countActiveByParent(run.parentRunId)
    if (activeCount >= this.maxChildren) {
      throw new Error(
        `Max concurrent sub-agents reached (${this.maxChildren}). Kill an existing one first.`,
      )
    }
    if (run.spawnDepth > this.maxSpawnDepth) {
      throw new Error(
        `Max spawn depth reached (${this.maxSpawnDepth}). Sub-agents at this depth cannot spawn children.`,
      )
    }
    this.runs.set(run.runId, run)
  }

  /** Get a run by ID. */
  get(runId: string): SubagentRun | undefined {
    return this.runs.get(runId)
  }

  /** Find a run by name (first active match). */
  findByName(name: string): SubagentRun | undefined {
    for (const run of this.runs.values()) {
      if (run.name === name && run.status === 'running') {
        return run
      }
    }
    return undefined
  }

  /** List all runs, optionally filtered by parent. */
  list(parentRunId?: string): SubagentRun[] {
    const all = Array.from(this.runs.values())
    if (parentRunId !== undefined) {
      return all.filter((r) => r.parentRunId === parentRunId)
    }
    return all
  }

  /** List active (running) runs. */
  listActive(parentRunId?: string): SubagentRun[] {
    return this.list(parentRunId).filter((r) => r.status === 'running')
  }

  /** Count active runs for a given parent. */
  countActiveByParent(parentRunId?: string): number {
    let count = 0
    for (const run of this.runs.values()) {
      if (run.status === 'running' && run.parentRunId === parentRunId) {
        count++
      }
    }
    return count
  }

  /** Mark a run as completed with a result. */
  complete(runId: string, result: string): void {
    const run = this.runs.get(runId)
    if (!run) return
    run.status = 'completed'
    run.result = result
    run.endedAt = Date.now()
  }

  /** Mark a run as failed with an error. */
  fail(runId: string, error: string): void {
    const run = this.runs.get(runId)
    if (!run) return
    run.status = 'error'
    run.error = error
    run.endedAt = Date.now()
  }

  /**
   * Abort a run and all its descendants (cascade kill).
   * Returns the number of runs aborted.
   */
  abort(runId: string): number {
    const run = this.runs.get(runId)
    if (!run) return 0
    let count = 0

    if (run.status === 'running') {
      run.abortController.abort()
      run.status = 'aborted'
      run.endedAt = Date.now()
      count++
    }

    // Cascade kill children
    for (const child of this.runs.values()) {
      if (child.parentRunId === runId && child.status === 'running') {
        count += this.abort(child.runId)
      }
    }

    return count
  }

  /** Clean up old finished runs (older than given ms). */
  sweep(maxAgeMs: number = 10 * 60 * 1000): number {
    const cutoff = Date.now() - maxAgeMs
    let removed = 0
    for (const [id, run] of this.runs.entries()) {
      if (run.endedAt && run.endedAt < cutoff) {
        this.runs.delete(id)
        removed++
      }
    }
    return removed
  }

  /** Serialize runs for inspection (strips AbortController). */
  toJSON(): Array<Omit<SubagentRun, 'abortController'>> {
    return Array.from(this.runs.values()).map(({ abortController: _, ...rest }) => rest)
  }
}

/**
 * @deprecated Use per-agent registry instead of the global singleton.
 * Kept for backward compatibility; falls back to a swept instance.
 */
let globalRegistry: SubagentRegistry | null = null
let globalSweepTimer: ReturnType<typeof setInterval> | null = null

const SWEEP_INTERVAL_MS = 5 * 60 * 1000

export function getGlobalRegistry(): SubagentRegistry {
  if (!globalRegistry) {
    globalRegistry = new SubagentRegistry()
    globalSweepTimer = setInterval(() => {
      globalRegistry?.sweep()
    }, SWEEP_INTERVAL_MS)
  }
  return globalRegistry
}

export function resetGlobalRegistry(): void {
  if (globalSweepTimer) {
    clearInterval(globalSweepTimer)
    globalSweepTimer = null
  }
  globalRegistry = null
}
