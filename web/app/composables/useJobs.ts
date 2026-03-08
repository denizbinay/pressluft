import { ref, readonly } from 'vue'
import type { CreateJobRequest, Job, JobEvent } from '~/lib/api-contract'
import type { JobStatus, JobTerminalStatus } from '~/lib/platform-contract.generated'
import { jobTerminalStatuses } from '~/lib/platform-contract.generated'
import { parseJob, parseJobEvents } from '~/lib/api-runtime'
export type { Job, JobEvent } from '~/lib/api-contract'

/** Connection mode for job monitoring */
export type ConnectionMode = 'streaming' | 'polling' | 'disconnected'

export function useJobs() {
  const { apiFetch, apiPath } = useApiClient()
  const activeJob = ref<Job | null>(null)
  const events = ref<JobEvent[]>([])
  const loading = ref(false)
  const error = ref('')
  const connectionMode = ref<ConnectionMode>('disconnected')

  const createJob = async (payload: CreateJobRequest) => {
    loading.value = true
    error.value = ''
    try {
      const job = parseJob(await apiFetch('/jobs', {
        method: 'POST',
        body: payload,
      }))
      activeJob.value = job
      return job
    } finally {
      loading.value = false
    }
  }

  const fetchJob = async (jobId: number) => {
    error.value = ''
    const job = parseJob(await apiFetch(`/jobs/${jobId}`))
    activeJob.value = job
    return job
  }

  const fetchJobEvents = async (jobId: number) => {
    error.value = ''
    const data = parseJobEvents(await apiFetch(`/jobs/${jobId}/events/history`))
    events.value = data
    return data
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

    const isTerminalStatus = (status: JobStatus) =>
      jobTerminalStatuses.includes(status as JobTerminalStatus)

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
      stream = new EventSource(apiPath(`/jobs/${jobId}/events`))
      updateMode('streaming')

      stream.addEventListener('job_event', (evt) => {
        try {
          const parsed = parseJobEvents([JSON.parse((evt as MessageEvent).data)])[0]
          events.value = [...events.value, parsed]
          onEvent?.(parsed)

          // Check for terminal events
          if (parsed.status && isTerminalStatus(parsed.status as JobStatus)) {
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
    fetchJobEvents,
    streamJobEvents,
    clearEvents,
  }
}
