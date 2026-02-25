<script setup lang="ts">
import { Card, CardContent } from "@/components/ui/card"
import type { Job } from '~/composables/useJobs'

const route = useRoute()
const router = useRouter()

const jobId = computed(() => {
  const id = Number(route.params.id)
  return Number.isNaN(id) ? null : id
})

const isInvalidId = computed(() => jobId.value === null)

const handleCompleted = (_job: Job) => {}

const handleFailed = (_job: Job, _error: string) => {}
</script>

<template>
  <div class="space-y-6">
    <!-- Error state for invalid job ID -->
    <template v-if="isInvalidId">
      <div>
        <h1 class="text-2xl font-semibold text-surface-50">Invalid Job ID</h1>
        <p class="mt-1 text-sm text-surface-400">
          The job ID provided is not valid.
        </p>
      </div>

      <Card
        class="rounded-xl border border-surface-800/60 bg-surface-900/50 backdrop-blur-sm py-0 shadow-none"
      >
        <CardContent class="px-6 py-5">
          <div class="flex flex-col items-center justify-center py-8 text-center">
            <div class="mb-4 rounded-full bg-danger-500/10 p-3">
              <svg
                class="h-6 w-6 text-danger-400"
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
            <p class="text-surface-300">Please provide a valid numeric job ID.</p>
          </div>
        </CardContent>
      </Card>
    </template>

    <!-- Valid job ID - show job details -->
    <template v-else>
      <div>
        <h1 class="text-2xl font-semibold text-surface-50">Job #{{ jobId }}</h1>
        <p class="mt-1 text-sm text-surface-400">
          Server provisioning progress
        </p>
      </div>

      <Card
        class="rounded-xl border border-surface-800/60 bg-surface-900/50 backdrop-blur-sm py-0 shadow-none"
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
        to="/settings?tab=servers"
        class="text-sm text-surface-400 transition-colors hover:text-surface-200"
      >
        &larr; Back to Servers
      </NuxtLink>
    </div>
  </div>
</template>
