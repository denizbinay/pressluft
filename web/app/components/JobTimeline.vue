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
  validate: "Validating request",
  provision: "Provisioning server",
  configure: "Configuring server",
  finalize: "Finalizing",
  delete: "Deleting server",
  rebuild: "Rebuilding server",
  resize: "Resizing server",
  update_firewalls: "Updating firewalls",
  manage_volume: "Managing volume",
}

const stepOrderByKind: Record<string, string[]> = {
  provision: ["validate", "provision", "configure", "finalize"],
  delete: ["validate", "delete", "finalize"],
  rebuild: ["validate", "rebuild", "finalize"],
  resize: ["validate", "resize", "finalize"],
  update_firewalls: ["validate", "update_firewalls", "finalize"],
  manage_volume: ["validate", "manage_volume", "finalize"],
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
  const rawKind = activeJob.value?.kind || ""
  const normalizedKind = rawKind.endsWith("_server")
    ? rawKind.slice(0, Math.max(rawKind.length - "_server".length, 0))
    : rawKind
  const eventStepOrder: string[] = []
  for (const event of events.value) {
    if (event.step_key && !eventStepOrder.includes(event.step_key)) {
      eventStepOrder.push(event.step_key)
    }
  }
  const stepOrder = stepOrderByKind[normalizedKind] || eventStepOrder
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

const jobKindLabel = computed(() => {
  const kind = activeJob.value?.kind
  return kind ? kind.replace(/_/g, " ") : ""
})

function formatPayloadValue(value: unknown): string {
  if (value === null) return "null"
  if (typeof value === "string") return value
  if (typeof value === "number" || typeof value === "boolean") return String(value)
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

const payloadSummary = computed(() => {
  const payload = activeJob.value?.payload
  if (!payload) return ""
  if (typeof payload === "string") {
    try {
      const parsed = JSON.parse(payload) as Record<string, unknown>
      const entries = Object.entries(parsed)
      if (entries.length === 0) return ""
      const summary = entries
        .map(([key, value]) => `${key}=${formatPayloadValue(value)}`)
        .join(", ")
      return summary.length > 160 ? `${summary.slice(0, 157)}...` : summary
    } catch {
      return payload
    }
  }
  return ""
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
      return "border-primary/30 bg-primary/10 text-primary"
    case "running":
    case "preparing":
    case "queued":
      return "border-accent/30 bg-accent/10 text-accent"
    case "failed":
    case "cancelled":
    case "timed_out":
      return "border-destructive/30 bg-destructive/10 text-destructive"
    default:
      return "border-border/60 bg-muted/60 text-foreground"
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
      <Spinner class="size-6 text-muted-foreground" />
    </div>

    <!-- Error state with retry -->
    <Alert
      v-else-if="connectionError"
      variant="destructive"
      :class="cn('border-destructive/30 bg-destructive/10 text-destructive')"
    >
      <div class="flex items-start justify-between gap-3">
        <div class="space-y-1">
          <AlertTitle class="text-sm font-medium text-destructive">Failed to load job</AlertTitle>
          <AlertDescription class="text-xs text-destructive/80">
            {{ connectionError }}
          </AlertDescription>
        </div>
        <Button
          variant="ghost"
          size="sm"
          class="shrink-0 h-auto rounded-md bg-destructive/10 px-3 py-1.5 text-xs font-medium text-destructive hover:bg-destructive/20"
          @click="retryLoad"
        >
          Retry
        </Button>
      </div>
    </Alert>

    <!-- Job timeline -->
    <template v-else>
      <!-- Job header -->
      <div class="flex items-center justify-between border-b border-border/40 pb-3">
        <div class="space-y-1">
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-foreground">Job #{{ jobId }}</span>
            <Badge variant="outline" :class="cn(statusBadgeClass(jobStatus))">
              {{ jobStatus }}
            </Badge>
            <!-- Connection mode indicator (only show for live view) -->
            <span
              v-if="!isHistoricalView && !isTerminal && connectionMode !== 'disconnected'"
              class="flex items-center gap-1 text-xs"
              :class="cn({
                'text-primary': connectionMode === 'streaming',
                'text-accent': connectionMode === 'polling',
              })"
            >
              <span
                class="h-1.5 w-1.5 rounded-full"
                :class="cn({
                  'bg-primary animate-pulse': connectionMode === 'streaming',
                  'bg-accent': connectionMode === 'polling',
                })"
              />
              {{ connectionMode === 'streaming' ? 'Live' : 'Polling' }}
            </span>
            <!-- Historical indicator -->
            <span
              v-if="isHistoricalView"
              class="flex items-center gap-1 text-xs text-muted-foreground"
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
          <div v-if="jobStartedAt || jobKindLabel || payloadSummary" class="space-y-0.5">
            <p v-if="jobStartedAt && !compact" class="text-xs text-muted-foreground">
              Started at {{ jobStartedAt }}
            </p>
            <p v-if="jobKindLabel" class="text-xs text-muted-foreground">
              Kind: <span class="font-medium text-foreground/80">{{ jobKindLabel }}</span>
            </p>
            <p v-if="payloadSummary" class="text-xs text-muted-foreground">
              Payload: <span class="font-mono text-foreground/70">{{ payloadSummary }}</span>
            </p>
          </div>
        </div>
        <NuxtLink
          v-if="compact"
          :to="`/jobs/${jobId}`"
          class="text-xs text-accent transition-colors hover:text-accent/80"
        >
          View Details &rarr;
        </NuxtLink>
      </div>

      <!-- Timeline steps -->
      <div class="relative pl-6">
        <!-- Vertical line -->
        <div class="absolute left-2.5 top-0 h-full w-px bg-border/60" />

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
                  'bg-muted text-muted-foreground': step.status === 'pending',
                  'bg-primary/20 text-primary': step.status === 'running',
                  'bg-primary/20 text-primary': step.status === 'completed',
                  'bg-destructive/20 text-destructive': step.status === 'failed',
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
                      'text-muted-foreground': step.status === 'pending',
                      'text-primary': step.status === 'running',
                      'text-foreground': step.status === 'completed',
                      'text-destructive': step.status === 'failed',
                    },
                    !isHistoricalView && 'transition-colors duration-200',
                  )"
                >
                  {{ step.label }}
                </span>
                <span v-if="step.timestamp" class="text-xs text-muted-foreground">
                  {{ formatTime(step.timestamp) }}
                </span>
              </div>
              <p
                v-if="step.message"
                class="mt-0.5 text-xs"
                :class="cn(
                  {
                    'text-muted-foreground': step.status !== 'failed',
                    'text-destructive/80': step.status === 'failed',
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
        :class="cn('mt-4 border-destructive/30 bg-destructive/10')"
      >
        <AlertTitle class="text-xs font-medium text-destructive">Error</AlertTitle>
        <AlertDescription class="mt-1 text-sm text-destructive">
          {{ activeJob.last_error }}
        </AlertDescription>
      </Alert>
    </template>
  </div>
</template>
