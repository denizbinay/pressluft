<script setup lang="ts">
import type { Job } from '~/composables/useJobs'

// Audit log entry - combines jobs and other account events
interface AuditEntry {
  id: string
  type: 'job' | 'server_created' | 'server_deleted' | 'provider_added' | 'provider_removed' | 'settings_changed'
  action: string
  target?: string
  status?: string
  timestamp: string
  details?: string
  jobId?: number
}

const entries = ref<AuditEntry[]>([])
const loading = ref(true)
const error = ref('')

// Fetch all jobs as audit entries (for now, jobs are our main audit source)
const fetchAuditLog = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await fetch('/api/jobs')
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(body.error || 'Failed to fetch activity log')
    }
    const jobs: Job[] = await res.json()
    
    // Convert jobs to audit entries
    entries.value = jobs.map((job) => ({
      id: `job-${job.id}`,
      type: 'job' as const,
      action: kindToAction(job.kind),
      target: job.server_id ? `Server #${job.server_id}` : undefined,
      status: job.status,
      timestamp: job.created_at,
      details: job.last_error,
      jobId: job.id,
    }))
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

const kindToAction = (kind: string): string => {
  const actions: Record<string, string> = {
    provision_server: 'Server provisioning',
    configure_server: 'Server configuration',
    deploy_site: 'Site deployment',
  }
  return actions[kind] || kind
}

const statusVariant = (status?: string): 'success' | 'warning' | 'danger' | 'default' => {
  if (!status) return 'default'
  if (status === 'succeeded') return 'success'
  if (status === 'failed' || status === 'cancelled' || status === 'timed_out') return 'danger'
  if (status === 'running' || status === 'preparing' || status === 'queued') return 'warning'
  return 'default'
}

const typeIcon = (type: AuditEntry['type']): string => {
  const icons: Record<string, string> = {
    job: 'M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z',
    server_created: 'M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2',
    server_deleted: 'M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16',
    provider_added: 'M12 4v16m8-8H4',
    provider_removed: 'M20 12H4',
    settings_changed: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35',
  }
  return icons[type] || icons.job
}

const formatDate = (iso: string): string => {
  try {
    const date = new Date(iso)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMs / 3600000)
    const diffDays = Math.floor(diffMs / 86400000)

    if (diffMins < 1) return 'Just now'
    if (diffMins < 60) return `${diffMins}m ago`
    if (diffHours < 24) return `${diffHours}h ago`
    if (diffDays < 7) return `${diffDays}d ago`

    return date.toLocaleDateString('en-US', {
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

onMounted(fetchAuditLog)
</script>

<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <p class="text-sm text-surface-400">
        Complete history of all actions performed in your account.
      </p>
      <button
        class="text-xs text-accent-400 hover:text-accent-300 transition-colors"
        :disabled="loading"
        @click="fetchAuditLog"
      >
        Refresh
      </button>
    </div>

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
        @click="fetchAuditLog"
      >
        Try again
      </button>
    </div>

    <!-- Empty state -->
    <div v-else-if="entries.length === 0" class="rounded-lg border border-dashed border-surface-700/50 px-4 py-8 text-center">
      <div class="flex justify-center">
        <svg class="h-10 w-10 text-surface-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </div>
      <h3 class="mt-3 text-sm font-medium text-surface-200">No activity yet</h3>
      <p class="mt-1 text-sm text-surface-500">
        Actions like server provisioning, site deployments, and configuration changes will appear here.
      </p>
    </div>

    <!-- Activity list -->
    <div v-else class="space-y-2">
      <div
        v-for="entry in entries"
        :key="entry.id"
        class="flex items-start gap-3 rounded-lg border border-surface-800/60 bg-surface-950/40 px-4 py-3"
      >
        <!-- Icon -->
        <div
          class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg"
          :class="{
            'bg-success-500/10 text-success-400': entry.status === 'succeeded',
            'bg-danger-500/10 text-danger-400': entry.status === 'failed' || entry.status === 'cancelled',
            'bg-warning-500/10 text-warning-400': entry.status === 'running' || entry.status === 'preparing' || entry.status === 'queued',
            'bg-surface-800/50 text-surface-400': !entry.status || !['succeeded', 'failed', 'cancelled', 'running', 'preparing', 'queued'].includes(entry.status),
          }"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" :d="typeIcon(entry.type)" />
          </svg>
        </div>

        <!-- Content -->
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-surface-200">{{ entry.action }}</span>
            <UiBadge v-if="entry.status" :variant="statusVariant(entry.status)" size="sm">
              {{ entry.status }}
            </UiBadge>
          </div>
          <p class="mt-0.5 text-xs text-surface-500">
            <span v-if="entry.target">{{ entry.target }} · </span>
            {{ formatDate(entry.timestamp) }}
          </p>
          <p v-if="entry.details" class="mt-1 text-xs text-danger-400/80">
            {{ entry.details }}
          </p>
        </div>

        <!-- Link to job details -->
        <NuxtLink
          v-if="entry.jobId"
          :to="`/jobs/${entry.jobId}`"
          class="shrink-0 text-xs text-accent-400 hover:text-accent-300 transition-colors"
        >
          Details
        </NuxtLink>
      </div>
    </div>

    <!-- Info note -->
    <div class="rounded-lg border border-surface-800/40 bg-surface-900/20 px-4 py-3">
      <div class="flex items-start gap-2">
        <svg class="mt-0.5 h-4 w-4 shrink-0 text-surface-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p class="text-xs text-surface-500">
          This audit log is permanent and cannot be deleted. All actions are recorded for compliance and troubleshooting purposes.
        </p>
      </div>
    </div>
  </div>
</template>
