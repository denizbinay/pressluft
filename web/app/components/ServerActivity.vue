<script setup lang="ts">
import type { Job } from '~/composables/useJobs'

interface Props {
  serverId: number
}

const props = defineProps<Props>()

const jobs = ref<Job[]>([])
const loading = ref(true)
const error = ref('')

const fetchServerJobs = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await fetch(`/api/servers/${props.serverId}/jobs`)
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(body.error || 'Failed to fetch server jobs')
    }
    jobs.value = await res.json()
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

const statusVariant = (status: string): 'success' | 'warning' | 'danger' | 'default' => {
  if (status === 'succeeded') return 'success'
  if (status === 'failed' || status === 'cancelled' || status === 'timed_out') return 'danger'
  if (status === 'running' || status === 'preparing' || status === 'queued') return 'warning'
  return 'default'
}

const formatDate = (iso: string): string => {
  try {
    return new Date(iso).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return iso
  }
}

const kindLabel = (kind: string): string => {
  const labels: Record<string, string> = {
    provision_server: 'Server Provisioning',
    configure_server: 'Server Configuration',
    deploy_site: 'Site Deployment',
  }
  return labels[kind] || kind
}

onMounted(fetchServerJobs)
</script>

<template>
  <div class="space-y-4">
    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-8">
      <svg class="h-5 w-5 animate-spin text-surface-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="rounded-lg border border-danger-600/30 bg-danger-900/20 px-4 py-3">
      <p class="text-sm text-danger-300">{{ error }}</p>
      <button
        class="mt-2 text-xs text-danger-400 hover:text-danger-300 transition-colors"
        @click="fetchServerJobs"
      >
        Try again
      </button>
    </div>

    <!-- Empty state -->
    <div v-else-if="jobs.length === 0" class="rounded-lg border border-dashed border-surface-700/50 px-4 py-8 text-center">
      <p class="text-sm text-surface-500">
        No activity recorded for this server yet.
      </p>
    </div>

    <!-- Jobs list -->
    <div v-else class="space-y-3">
      <div
        v-for="job in jobs"
        :key="job.id"
        class="flex items-center justify-between rounded-lg border border-surface-800/60 bg-surface-950/40 px-4 py-3"
      >
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-surface-200">{{ kindLabel(job.kind) }}</span>
            <UiBadge :variant="statusVariant(job.status)" size="sm">{{ job.status }}</UiBadge>
          </div>
          <p class="mt-0.5 text-xs text-surface-500">
            {{ formatDate(job.created_at) }}
            <span v-if="job.last_error" class="text-danger-400"> · {{ job.last_error }}</span>
          </p>
        </div>
        <NuxtLink
          :to="`/jobs/${job.id}`"
          class="shrink-0 text-xs text-accent-400 hover:text-accent-300 transition-colors"
        >
          View Details
        </NuxtLink>
      </div>
    </div>
  </div>
</template>
