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

export function useJobs() {
  const activeJob = ref<Job | null>(null)
  const events = ref<JobEvent[]>([])
  const loading = ref(false)
  const error = ref('')

  const createJob = async (payload: { kind?: string, server_id?: number }) => {
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

  const streamJobEvents = (jobId: number, onEvent?: (event: JobEvent) => void) => {
    const stream = new EventSource(`/api/jobs/${jobId}/events`)
    stream.addEventListener('job_event', (evt) => {
      try {
        const parsed = JSON.parse((evt as MessageEvent).data) as JobEvent
        events.value = [...events.value, parsed]
        onEvent?.(parsed)
      } catch {
        // Ignore malformed event payloads to keep stream alive.
      }
    })

    stream.onerror = () => {
      error.value = 'Live job stream disconnected'
    }

    return () => stream.close()
  }

  return {
    activeJob: readonly(activeJob),
    events: readonly(events),
    loading: readonly(loading),
    error: readonly(error),
    createJob,
    fetchJob,
    streamJobEvents,
  }
}
