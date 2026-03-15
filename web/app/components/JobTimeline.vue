<script setup lang="ts">
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Spinner } from "@/components/ui/spinner"
import { cn } from "@/lib/utils"
import type { Job } from "~/composables/useJobs"
import { useTimelineSteps } from "~/composables/useTimelineSteps"
import { useJobStream } from "~/composables/useJobStream"
import {
  jobTerminalStatuses,
  type JobStatus,
  type JobTerminalStatus,
} from "~/lib/platform-contract.generated"

interface Props {
  jobId: string
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

const {
  activeJob,
  events,
  connectionMode,
  isHistoricalView,
  loading,
  connectionError,
  retryLoad,
} = useJobStream({
  jobId: props.jobId,
  autoConnect: props.autoConnect,
  onCompleted: (job) => emit("completed", job),
  onFailed: (job, error) => emit("failed", job, error),
})

const { steps, jobKindLabel, payloadSummary } = useTimelineSteps(activeJob, events)

// Job metadata
const jobStatus = computed(() => activeJob.value?.status || "unknown")
const jobStartedAt = computed(() => {
  if (!activeJob.value?.created_at) return ""
  return formatTime(activeJob.value.created_at)
})

const isTerminal = computed(() => {
  const status = activeJob.value?.status
  return status ? jobTerminalStatuses.includes(status as JobTerminalStatus) : false
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

function statusBadgeClass(status: JobStatus | "unknown") {
  switch (status) {
    case "succeeded":
      return "border-primary/30 bg-primary/10 text-primary"
    case "running":
    case "queued":
      return "border-accent/30 bg-accent/10 text-accent"
    case "failed":
      return "border-destructive/30 bg-destructive/10 text-destructive"
    default:
      return "border-border/60 bg-muted/60 text-foreground"
  }
}
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
                  'bg-primary/20 text-primary': step.status === 'running' || step.status === 'completed',
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
