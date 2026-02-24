import { ref, readonly } from 'vue'

export interface Job {
  id: number
  server_id?: number
  kind: string
  status: string
  current_step: string
  retry_count: number
  last_error?: string
  created_at: string
  updated_at: string
}

export interface JobEvent {
  job_id: number
  seq: number
  event_type: string
  level: string
  step_key?: string
  status?: string
  message: string
  payload?: string
  occurred_at: string
}

/** Connection mode for job monitoring */
export type ConnectionMode = 'streaming' | 'polling' | 'disconnected'

export function useJobs() {
  const activeJob = ref<Job | null>(null)
  const events = ref<JobEvent[]>([])
  const loading = ref(false)
  const error = ref('')
  const connectionMode = ref<ConnectionMode>('disconnected')

  const createJob = async (payload: { kind?: string; server_id?: number }) => {
    loading.value = true
    error.value = ''
    try {
      const res = await fetch('/api/jobs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: res.statusText }))
        throw new Error(body.error || 'Failed to create job')
      }
      const job = await res.json()
      activeJob.value = job
      return job as Job
    } finally {
      loading.value = false
    }
  }

  const fetchJob = async (jobId: number) => {
    error.value = ''
    const res = await fetch(`/api/jobs/${jobId}`)
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(body.error || 'Failed to fetch job')
    }
    const job = await res.json()
    activeJob.value = job
    return job as Job
  }

  /**
   * Stream job events via SSE with automatic polling fallback.
   * If SSE fails, falls back to polling the job status every 2 seconds.
   */
  const streamJobEvents = (
    jobId: number,
    onEvent?: (event: JobEvent) => void,
    onModeChange?: (mode: ConnectionMode) => void,
  ) => {
    let stream: EventSource | null = null
    let pollInterval: ReturnType<typeof setInterval> | null = null
    let closed = false

    const updateMode = (mode: ConnectionMode) => {
      connectionMode.value = mode
      onModeChange?.(mode)
    }

    const isTerminalStatus = (status: string) =>
      ['succeeded', 'failed', 'cancelled', 'timed_out'].includes(status)

    // Start polling fallback
    const startPolling = () => {
      if (pollInterval || closed) return

      updateMode('polling')
      pollInterval = setInterval(async () => {
        if (closed) {
          if (pollInterval) clearInterval(pollInterval)
          return
        }

        try {
          const job = await fetchJob(jobId)

          // Synthesize a step event from job state for UI updates
          if (job.current_step) {
            const syntheticEvent: JobEvent = {
              job_id: jobId,
              seq: Date.now(), // Use timestamp as pseudo-sequence
              event_type: 'step_update',
              level: 'info',
              step_key: job.current_step,
              status: job.status === 'running' ? 'running' : job.status,
              message: `Step: ${job.current_step}`,
              occurred_at: job.updated_at,
            }

            // Only add if we don't have this step yet
            const hasStep = events.value.some(
              (e) => e.step_key === job.current_step && e.status === syntheticEvent.status,
            )
            if (!hasStep) {
              events.value = [...events.value, syntheticEvent]
              onEvent?.(syntheticEvent)
            }
          }

          // Stop polling on terminal status
          if (isTerminalStatus(job.status)) {
            if (pollInterval) {
              clearInterval(pollInterval)
              pollInterval = null
            }
            updateMode('disconnected')
          }
        } catch {
          // Continue polling even on errors
        }
      }, 2000)
    }

    // Try SSE first
    try {
      stream = new EventSource(`/api/jobs/${jobId}/events`)
      updateMode('streaming')

      stream.addEventListener('job_event', (evt) => {
        try {
          const parsed = JSON.parse((evt as MessageEvent).data) as JobEvent
          events.value = [...events.value, parsed]
          onEvent?.(parsed)

          // Check for terminal events
          if (parsed.status && isTerminalStatus(parsed.status)) {
            fetchJob(jobId).catch(() => {})
          }
        } catch {
          // Ignore malformed event payloads
        }
      })

      stream.onerror = () => {
        // SSE failed - close and fall back to polling
        error.value = ''
        stream?.close()
        stream = null
        startPolling()
      }
    } catch {
      // SSE not supported or failed to create - use polling
      startPolling()
    }

    // Return cleanup function
    return () => {
      closed = true
      if (stream) {
        stream.close()
        stream = null
      }
      if (pollInterval) {
        clearInterval(pollInterval)
        pollInterval = null
      }
      updateMode('disconnected')
    }
  }

  const clearEvents = () => {
    events.value = []
  }

  return {
    activeJob: readonly(activeJob),
    events: readonly(events),
    loading: readonly(loading),
    error: readonly(error),
    connectionMode: readonly(connectionMode),
    createJob,
    fetchJob,
    streamJobEvents,
    clearEvents,
  }
}
