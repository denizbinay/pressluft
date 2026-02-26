<script setup lang="ts">
import { Card, CardContent } from "@/components/ui/card"
import type { Job } from '~/composables/useJobs'
import { useJobs } from "~/composables/useJobs"

const route = useRoute()
const router = useRouter()
const { fetchJob } = useJobs()

const jobId = computed(() => {
  const id = Number(route.params.id)
  return Number.isNaN(id) ? null : id
})

const isInvalidId = computed(() => jobId.value === null)

const job = ref<Job | null>(null)
const jobError = ref("")

const jobKindLabel = (kind: string): string => {
  const labels: Record<string, string> = {
    provision_server: "Server provisioning",
    delete_server: "Server deletion",
    rebuild_server: "Server rebuild",
    resize_server: "Server resize",
    update_firewalls: "Firewall update",
    manage_volume: "Volume management",
  }
  return labels[kind] || kind
}

const jobSubtitle = computed(() => {
  if (!job.value?.kind) return "Job progress"
  return `${jobKindLabel(job.value.kind)} progress`
})

const handleCompleted = (_job: Job) => {}

const handleFailed = (_job: Job, _error: string) => {}

onMounted(async () => {
  if (!jobId.value) return
  jobError.value = ""
  try {
    job.value = await fetchJob(jobId.value)
  } catch (e: any) {
    jobError.value = e.message || "Failed to load job"
  }
})
</script>

<template>
  <div class="space-y-6">
    <!-- Error state for invalid job ID -->
    <template v-if="isInvalidId">
      <div class="space-y-2">
        <div class="flex items-center gap-2 text-xs text-muted-foreground">
          <NuxtLink to="/activity" class="transition-colors hover:text-foreground">
            Activity
          </NuxtLink>
          <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
          <span class="text-foreground">Job</span>
        </div>
        <div>
          <h1 class="text-2xl font-semibold text-foreground">Invalid Job ID</h1>
          <p class="mt-1 text-sm text-muted-foreground">
            The job ID provided is not valid.
          </p>
        </div>
      </div>

      <Card
        class="rounded-xl border border-border/60 bg-card/50 backdrop-blur-sm py-0 shadow-none"
      >
        <CardContent class="px-6 py-5">
          <div class="flex flex-col items-center justify-center py-8 text-center">
            <div class="mb-4 rounded-full bg-destructive/10 p-3">
              <svg
                class="h-6 w-6 text-destructive"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                />
              </svg>
            </div>
            <p class="text-muted-foreground">Please provide a valid numeric job ID.</p>
          </div>
        </CardContent>
      </Card>
    </template>

    <!-- Valid job ID - show job details -->
    <template v-else>
      <div class="space-y-2">
        <div class="flex items-center gap-2 text-xs text-muted-foreground">
          <NuxtLink to="/activity" class="transition-colors hover:text-foreground">
            Activity
          </NuxtLink>
          <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
          <template v-if="job?.server_id">
            <NuxtLink
              :to="`/servers/${job.server_id}`"
              class="transition-colors hover:text-foreground"
            >
              Server #{{ job.server_id }}
            </NuxtLink>
            <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </template>
          <span class="text-foreground">Job #{{ jobId }}</span>
        </div>
        <div>
          <h1 class="text-2xl font-semibold text-foreground">Job #{{ jobId }}</h1>
          <p class="mt-1 text-sm text-muted-foreground">
            {{ jobSubtitle }}
          </p>
          <p v-if="jobError" class="mt-1 text-xs text-destructive">
            {{ jobError }}
          </p>
        </div>
      </div>

      <Card
        class="rounded-xl border border-border/60 bg-card/50 backdrop-blur-sm py-0 shadow-none"
      >
        <CardContent class="px-6 py-5">
          <JobTimeline
            :job-id="jobId!"
            @completed="handleCompleted"
            @failed="handleFailed"
          />
        </CardContent>
      </Card>
    </template>

    <div class="flex gap-2">
      <NuxtLink
        to="/activity"
        class="text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        &larr; Back to Activity
      </NuxtLink>
    </div>
  </div>
</template>
