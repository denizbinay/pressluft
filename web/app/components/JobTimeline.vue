<script setup lang="ts">
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Spinner } from "@/components/ui/spinner"
import { cn } from "@/lib/utils"
import { useJobs, type Job, type JobEvent, type ConnectionMode } from "~/composables/useJobs"

interface Props {
  jobId: number
  autoConnect?: boolean
  /** Compact mode for embedding in modals */
  compact?: boolean
}

interface Emits {
  (e: "completed", job: Job): void
  (e: "failed", job: Job, error: string): void
}

const props = withDefaults(defineProps<Props>(), {
  autoConnect: true,
  compact: false,
})

const emit = defineEmits<Emits>()

const { activeJob, events, connectionMode, fetchJob, fetchJobEvents, streamJobEvents, clearEvents } = useJobs()

/** Whether we're showing a historical (already completed) job vs live */
const isHistoricalView = ref(false)

const loading = ref(true)
const connectionError = ref("")
const retryCount = ref(0)

// Step key to human-readable label mapping (matches backend executor steps)
const stepLabels: Record<string, string> = {
  validate: "Validating configuration",
  create_ssh_key: "Creating SSH key",
  create_server: "Creating server",
  wait_running: "Waiting for server",
  finalize: "Finalizing setup",
}

// Derive steps from events
interface TimelineStep {
  key: string
  label: string
  status: "pending" | "running" | "completed" | "failed"
  message?: string
  timestamp?: string
}

const steps = computed<TimelineStep[]>(() => {
  // Step order matches backend executor (internal/worker/executor.go)
  const stepOrder = ["validate", "create_ssh_key", "create_server", "wait_running", "finalize"]
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

    let status: TimelineStep["status"] = "pending"
    if (latestEvent) {
      if (latestEvent.status === "completed" || latestEvent.event_type === "step_completed") {
        status = "completed"
      } else if (latestEvent.status === "failed" || latestEvent.event_type === "step_failed") {
        status = "failed"
      } else if (latestEvent.status === "running" || latestEvent.event_type === "step_started") {
        status = "running"
      }
    }

    // Check if this is the current step from the job
    if (activeJob.value?.current_step === key && status === "pending") {
      status = "running"
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
const jobStatus = computed(() => activeJob.value?.status || "unknown")
const jobStartedAt = computed(() => {
  if (!activeJob.value?.created_at) return ""
  return formatTime(activeJob.value.created_at)
})

const isTerminal = computed(() => {
  const status = activeJob.value?.status
  return status === "succeeded" || status === "failed" || status === "cancelled" || status === "timed_out"
})

// Format timestamp to readable time
function formatTime(iso: string): string {
  try {
    return new Date(iso).toLocaleTimeString("en-US", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    })
  } catch {
    return iso
  }
}

function statusBadgeClass(status: string) {
  switch (status) {
    case "succeeded":
      return "border-success-700/40 bg-success-900/40 text-success-300"
    case "running":
    case "preparing":
    case "queued":
      return "border-warning-700/40 bg-warning-900/40 text-warning-300"
    case "failed":
    case "cancelled":
    case "timed_out":
      return "border-danger-700/40 bg-danger-900/40 text-danger-300"
    default:
      return "border-surface-700/60 bg-surface-800/60 text-surface-100"
  }
}

// SSE cleanup function
let closeStream: (() => void) | null = null

// Watch for terminal states and emit events
watch(
  () => activeJob.value?.status,
  (status) => {
    if (!activeJob.value) return

    if (status === "succeeded") {
      emit("completed", activeJob.value)
    } else if (status === "failed" || status === "cancelled" || status === "timed_out") {
      emit("failed", activeJob.value, activeJob.value.last_error || "Unknown error")
    }
  },
)

// Handle event updates to refresh job status
function handleEvent(event: JobEvent) {
  // Refresh job when we get terminal events
  if (event.event_type === "job_completed" || event.event_type === "job_failed") {
    fetchJob(props.jobId).catch(() => {})
  }
}

// Handle connection mode changes
function handleModeChange(mode: ConnectionMode) {
  if (mode === "polling") {
    // Clear any previous connection error when we successfully fall back to polling
    connectionError.value = ""
  }
}

// Retry loading the job
async function retryLoad() {
  retryCount.value++
  connectionError.value = ""
  loading.value = true
  clearEvents()

  try {
    const job = await fetchJob(props.jobId)

    // Check if job is already in terminal state (historical view)
    const terminalStatuses = ["succeeded", "failed", "cancelled", "timed_out"]
    if (terminalStatuses.includes(job.status)) {
      isHistoricalView.value = true
      await fetchJobEvents(props.jobId)
      loading.value = false
      return
    }

    loading.value = false
    if (props.autoConnect) {
      if (closeStream) closeStream()
      closeStream = streamJobEvents(props.jobId, handleEvent, handleModeChange)
    }
  } catch (e: any) {
    connectionError.value = e.message || "Failed to load job"
    loading.value = false
  }
}

onMounted(async () => {
  try {
    const job = await fetchJob(props.jobId)

    // Check if job is already in terminal state (historical view)
    const terminalStatuses = ["succeeded", "failed", "cancelled", "timed_out"]
    if (terminalStatuses.includes(job.status)) {
      // Historical view: fetch all events at once, no streaming
      isHistoricalView.value = true
      await fetchJobEvents(props.jobId)
      loading.value = false
      return
    }

    // Live view: stream events
    loading.value = false
    if (props.autoConnect) {
      closeStream = streamJobEvents(props.jobId, handleEvent, handleModeChange)
    }
  } catch (e: any) {
    connectionError.value = e.message || "Failed to load job"
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
      <Spinner class="size-6 text-surface-400" />
    </div>

    <!-- Error state with retry -->
    <Alert
      v-else-if="connectionError"
      variant="destructive"
      :class="cn('border-danger-600/30 bg-danger-900/20 text-danger-300')"
    >
      <div class="flex items-start justify-between gap-3">
        <div class="space-y-1">
          <AlertTitle class="text-sm font-medium text-danger-300">Failed to load job</AlertTitle>
          <AlertDescription class="text-xs text-danger-400/80">
            {{ connectionError }}
          </AlertDescription>
        </div>
        <Button
          variant="ghost"
          size="sm"
          class="shrink-0 h-auto rounded-md bg-danger-800/50 px-3 py-1.5 text-xs font-medium text-danger-200 hover:bg-danger-800/70"
          @click="retryLoad"
        >
          Retry
        </Button>
      </div>
    </Alert>

    <!-- Job timeline -->
    <template v-else>
      <!-- Job header -->
      <div class="flex items-center justify-between border-b border-surface-800/40 pb-3">
        <div class="space-y-1">
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-surface-200">Job #{{ jobId }}</span>
            <Badge variant="outline" :class="cn(statusBadgeClass(jobStatus))">
              {{ jobStatus }}
            </Badge>
            <!-- Connection mode indicator (only show for live view) -->
            <span
              v-if="!isHistoricalView && !isTerminal && connectionMode !== 'disconnected'"
              class="flex items-center gap-1 text-xs"
              :class="cn({
                'text-success-500': connectionMode === 'streaming',
                'text-warning-500': connectionMode === 'polling',
              })"
            >
              <span
                class="h-1.5 w-1.5 rounded-full"
                :class="cn({
                  'bg-success-500 animate-pulse': connectionMode === 'streaming',
                  'bg-warning-500': connectionMode === 'polling',
                })"
              />
              {{ connectionMode === 'streaming' ? 'Live' : 'Polling' }}
            </span>
            <!-- Historical indicator -->
            <span
              v-if="isHistoricalView"
              class="flex items-center gap-1 text-xs text-surface-500"
            >
              <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              Completed
            </span>
          </div>
          <p v-if="jobStartedAt && !compact" class="text-xs text-surface-500">
            Started at {{ jobStartedAt }}
          </p>
        </div>
        <NuxtLink
          v-if="compact"
          :to="`/jobs/${jobId}`"
          class="text-xs text-accent-400 transition-colors hover:text-accent-300"
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
              class="absolute -left-3.5 flex h-5 w-5 items-center justify-center rounded-full"
              :class="cn(
                {
                  'bg-surface-800 text-surface-500': step.status === 'pending',
                  'bg-primary-700/50 text-primary-300': step.status === 'running',
                  'bg-success-900/60 text-success-400': step.status === 'completed',
                  'bg-danger-900/60 text-danger-400': step.status === 'failed',
                },
                !isHistoricalView && 'transition-all duration-300',
              )"
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

              <!-- Running: spinner (only animate if live view) -->
              <Spinner
                v-else-if="step.status === 'running'"
                :class="cn('size-3', isHistoricalView && 'animate-none')"
              />

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
                  class="text-sm font-medium"
                  :class="cn(
                    {
                      'text-surface-500': step.status === 'pending',
                      'text-primary-300': step.status === 'running',
                      'text-surface-200': step.status === 'completed',
                      'text-danger-400': step.status === 'failed',
                    },
                    !isHistoricalView && 'transition-colors duration-200',
                  )"
                >
                  {{ step.label }}
                </span>
                <span v-if="step.timestamp" class="text-xs text-surface-600">
                  {{ formatTime(step.timestamp) }}
                </span>
              </div>
              <p
                v-if="step.message"
                class="mt-0.5 text-xs"
                :class="cn(
                  {
                    'text-surface-500': step.status !== 'failed',
                    'text-danger-400/80': step.status === 'failed',
                  },
                  !isHistoricalView && 'transition-opacity duration-200',
                )"
              >
                {{ step.message }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Error message for failed jobs -->
      <Alert
        v-if="activeJob?.status === 'failed' && activeJob.last_error"
        variant="destructive"
        :class="cn('mt-4 border-danger-600/30 bg-danger-900/20')"
      >
        <AlertTitle class="text-xs font-medium text-danger-400">Error</AlertTitle>
        <AlertDescription class="mt-1 text-sm text-danger-300">
          {{ activeJob.last_error }}
        </AlertDescription>
      </Alert>
    </template>
  </div>
</template>
