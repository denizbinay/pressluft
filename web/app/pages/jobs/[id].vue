<script setup lang="ts">
const route = useRoute()

const jobId = computed(() => {
  const id = Number(route.params.id)
  return Number.isNaN(id) ? null : id
})

const isInvalidId = computed(() => jobId.value === null)
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

      <UiCard>
        <div class="flex flex-col items-center justify-center py-8 text-center">
          <div class="mb-4 rounded-full bg-red-500/10 p-3">
            <svg
              class="h-6 w-6 text-red-400"
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
      </UiCard>
    </template>

    <!-- Valid job ID - show job details -->
    <template v-else>
      <div>
        <h1 class="text-2xl font-semibold text-surface-50">Job #{{ jobId }}</h1>
        <p class="mt-1 text-sm text-surface-400">
          Server provisioning progress
        </p>
      </div>

      <UiCard>
        <!-- JobTimeline component will be integrated here -->
        <!-- Will be replaced with: <JobTimeline :job-id="jobId" /> -->
        <div class="py-4 text-center">
          <div class="mb-3 inline-flex h-8 w-8 animate-spin items-center justify-center rounded-full border-2 border-surface-700 border-t-primary-500" />
          <p class="text-surface-400">Loading job {{ jobId }}...</p>
        </div>
      </UiCard>
    </template>

    <div class="flex gap-2">
      <NuxtLink
        to="/servers"
        class="text-sm text-surface-400 transition-colors hover:text-surface-200"
      >
        &larr; Back to Servers
      </NuxtLink>
    </div>
  </div>
</template>
