<script setup lang="ts">
import { useJobs, type Job, type JobEvent, type ConnectionMode } from '~/composables/useJobs'

interface Props {
  jobId: number
  autoConnect?: boolean
  /** Compact mode for embedding in modals */
  compact?: boolean
}

interface Emits {
  (e: 'completed', job: Job): void
  (e: 'failed', job: Job, error: string): void
}

const props = withDefaults(defineProps<Props>(), {
  autoConnect: true,
  compact: false,
})

const emit = defineEmits<Emits>()

const { activeJob, events, connectionMode, fetchJob, streamJobEvents, clearEvents } = useJobs()

const loading = ref(true)
const connectionError = ref('')
const retryCount = ref(0)

// Step key to human-readable label mapping (matches backend executor steps)
const stepLabels: Record<string, string> = {
  validate: 'Validating configuration',
  create_ssh_key: 'Creating SSH key',
  create_server: 'Creating server',
  wait_running: 'Waiting for server',
  finalize: 'Finalizing setup',
}

// Derive steps from events
interface TimelineStep {
  key: string
  label: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  message?: string
  timestamp?: string
}

const steps = computed<TimelineStep[]>(() => {
  // Step order matches backend executor (internal/worker/executor.go)
  const stepOrder = ['validate', 'create_ssh_key', 'create_server', 'wait_running', 'finalize']
  const eventsByStep = new Map<string, JobEvent[]>()

  // Group events by step_key
  for (const event of events.value) {
    if (event.step_key) {
      const existing = eventsByStep.get(event.step_key) || []
      existing.push(event)
      eventsByStep.set(event.step_key, existing)
    }
  }

  // Build timeline steps
  return stepOrder.map((key) => {
    const stepEvents = eventsByStep.get(key) || []
    const latestEvent = stepEvents[stepEvents.length - 1]

    let status: TimelineStep['status'] = 'pending'
    if (latestEvent) {
      if (latestEvent.status === 'completed' || latestEvent.event_type === 'step_completed') {
        status = 'completed'
      } else if (latestEvent.status === 'failed' || latestEvent.event_type === 'step_failed') {
        status = 'failed'
      } else if (latestEvent.status === 'running' || latestEvent.event_type === 'step_started') {
        status = 'running'
      }
    }

    // Check if this is the current step from the job
    if (activeJob.value?.current_step === key && status === 'pending') {
      status = 'running'
    }

    return {
      key,
      label: stepLabels[key] || key,
      status,
      message: latestEvent?.message,
      timestamp: latestEvent?.occurred_at,
    }
  })
})

// Job metadata
const jobStatus = computed(() => activeJob.value?.status || 'unknown')
const jobStartedAt = computed(() => {
  if (!activeJob.value?.created_at) return ''
  return formatTime(activeJob.value.created_at)
})

const isTerminal = computed(() => {
  const status = activeJob.value?.status
  return status === 'succeeded' || status === 'failed' || status === 'cancelled' || status === 'timed_out'
})

// Format timestamp to readable time
function formatTime(iso: string): string {
  try {
    return new Date(iso).toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  } catch {
    return iso
  }
}

// Status badge variant
function statusVariant(status: string): 'success' | 'warning' | 'danger' | 'default' {
  switch (status) {
    case 'succeeded':
      return 'success'
    case 'running':
    case 'preparing':
    case 'queued':
      return 'warning'
    case 'failed':
    case 'cancelled':
    case 'timed_out':
      return 'danger'
    default:
      return 'default'
  }
}

// SSE cleanup function
let closeStream: (() => void) | null = null

// Watch for terminal states and emit events
watch(
  () => activeJob.value?.status,
  (status) => {
    if (!activeJob.value) return

    if (status === 'succeeded') {
      emit('completed', activeJob.value)
    } else if (status === 'failed' || status === 'cancelled' || status === 'timed_out') {
      emit('failed', activeJob.value, activeJob.value.last_error || 'Unknown error')
    }
  },
)

// Handle event updates to refresh job status
function handleEvent(event: JobEvent) {
  // Refresh job when we get terminal events
  if (event.event_type === 'job_completed' || event.event_type === 'job_failed') {
    fetchJob(props.jobId).catch(() => {})
  }
}

// Handle connection mode changes
function handleModeChange(mode: ConnectionMode) {
  if (mode === 'polling') {
    // Clear any previous connection error when we successfully fall back to polling
    connectionError.value = ''
  }
}

// Retry loading the job
async function retryLoad() {
  retryCount.value++
  connectionError.value = ''
  loading.value = true
  clearEvents()

  try {
    await fetchJob(props.jobId)
    loading.value = false

    if (props.autoConnect && !isTerminal.value) {
      if (closeStream) closeStream()
      closeStream = streamJobEvents(props.jobId, handleEvent, handleModeChange)
    }
  } catch (e: any) {
    connectionError.value = e.message || 'Failed to load job'
    loading.value = false
  }
}

onMounted(async () => {
  try {
    await fetchJob(props.jobId)
    loading.value = false

    if (props.autoConnect && !isTerminal.value) {
      closeStream = streamJobEvents(props.jobId, handleEvent, handleModeChange)
    }
  } catch (e: any) {
    connectionError.value = e.message || 'Failed to load job'
    loading.value = false
  }
})

onUnmounted(() => {
  if (closeStream) {
    closeStream()
    closeStream = null
  }
})
</script>

<template>
  <div class="space-y-4">
    <!-- Loading state -->
    <div v-if="loading" class="flex items-center justify-center py-8">
      <svg
        class="h-6 w-6 animate-spin text-surface-400"
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
    </div>

    <!-- Error state with retry -->
    <div
      v-else-if="connectionError"
      class="rounded-lg border border-danger-600/30 bg-danger-900/20 px-4 py-3"
    >
      <div class="flex items-start justify-between gap-3">
        <div class="space-y-1">
          <p class="text-sm font-medium text-danger-300">Failed to load job</p>
          <p class="text-xs text-danger-400/80">{{ connectionError }}</p>
        </div>
        <button
          class="shrink-0 rounded-md bg-danger-800/50 px-3 py-1.5 text-xs font-medium text-danger-200 transition-colors hover:bg-danger-800/70"
          @click="retryLoad"
        >
          Retry
        </button>
      </div>
    </div>

    <!-- Job timeline -->
    <template v-else>
      <!-- Job header -->
      <div class="flex items-center justify-between border-b border-surface-800/40 pb-3">
        <div class="space-y-1">
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-surface-200">Job #{{ jobId }}</span>
            <UiBadge :variant="statusVariant(jobStatus)">{{ jobStatus }}</UiBadge>
            <!-- Connection mode indicator (only show when not in terminal state) -->
            <span
              v-if="!isTerminal && connectionMode !== 'disconnected'"
              class="flex items-center gap-1 text-xs"
              :class="{
                'text-success-500': connectionMode === 'streaming',
                'text-warning-500': connectionMode === 'polling',
              }"
            >
              <span
                class="h-1.5 w-1.5 rounded-full"
                :class="{
                  'bg-success-500 animate-pulse': connectionMode === 'streaming',
                  'bg-warning-500': connectionMode === 'polling',
                }"
              />
              {{ connectionMode === 'streaming' ? 'Live' : 'Polling' }}
            </span>
          </div>
          <p v-if="jobStartedAt && !compact" class="text-xs text-surface-500">Started at {{ jobStartedAt }}</p>
        </div>
        <NuxtLink
          v-if="compact"
          :to="`/jobs/${jobId}`"
          class="text-xs text-accent-400 hover:text-accent-300 transition-colors"
        >
          View Details &rarr;
        </NuxtLink>
      </div>

      <!-- Timeline steps -->
      <div class="relative pl-6">
        <!-- Vertical line -->
        <div class="absolute left-2.5 top-0 h-full w-px bg-surface-800/60" />

        <div class="space-y-4">
          <div
            v-for="(step, index) in steps"
            :key="step.key"
            class="relative flex items-start gap-3"
          >
            <!-- Step indicator -->
            <div
              class="absolute -left-3.5 flex h-5 w-5 items-center justify-center rounded-full transition-all duration-300"
              :class="{
                'bg-surface-800 text-surface-500': step.status === 'pending',
                'bg-primary-700/50 text-primary-300': step.status === 'running',
                'bg-success-900/60 text-success-400': step.status === 'completed',
                'bg-danger-900/60 text-danger-400': step.status === 'failed',
              }"
            >
              <!-- Pending: empty circle -->
              <svg
                v-if="step.status === 'pending'"
                class="h-2 w-2"
                fill="currentColor"
                viewBox="0 0 8 8"
              >
                <circle cx="4" cy="4" r="3" />
              </svg>

              <!-- Running: spinner -->
              <svg
                v-else-if="step.status === 'running'"
                class="h-3 w-3 animate-spin"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>

              <!-- Completed: checkmark -->
              <svg
                v-else-if="step.status === 'completed'"
                class="h-3 w-3"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="3"
              >
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
              </svg>

              <!-- Failed: X -->
              <svg
                v-else-if="step.status === 'failed'"
                class="h-3 w-3"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="3"
              >
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </div>

            <!-- Step content -->
            <div class="min-w-0 flex-1 pt-0.5">
              <div class="flex items-center gap-2">
                <span
                  class="text-sm font-medium transition-colors duration-200"
                  :class="{
                    'text-surface-500': step.status === 'pending',
                    'text-primary-300': step.status === 'running',
                    'text-surface-200': step.status === 'completed',
                    'text-danger-400': step.status === 'failed',
                  }"
                >
                  {{ step.label }}
                </span>
                <span v-if="step.timestamp" class="text-xs text-surface-600">
                  {{ formatTime(step.timestamp) }}
                </span>
              </div>
              <p
                v-if="step.message"
                class="mt-0.5 text-xs transition-opacity duration-200"
                :class="{
                  'text-surface-500': step.status !== 'failed',
                  'text-danger-400/80': step.status === 'failed',
                }"
              >
                {{ step.message }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Error message for failed jobs -->
      <div
        v-if="activeJob?.status === 'failed' && activeJob.last_error"
        class="mt-4 rounded-lg border border-danger-600/30 bg-danger-900/20 px-4 py-3"
      >
        <p class="text-xs font-medium text-danger-400">Error</p>
        <p class="mt-1 text-sm text-danger-300">{{ activeJob.last_error }}</p>
      </div>
    </template>
  </div>
</template>
