<script setup lang="ts">
import type { Job } from "~/composables/useJobs"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { Item, ItemActions, ItemContent, ItemMedia, ItemTitle } from "@/components/ui/item"
import { Spinner } from "@/components/ui/spinner"
import { cn } from "@/lib/utils"

// Audit log entry - combines jobs and other account events
interface AuditEntry {
  id: string
  type:
    | "job"
    | "server_created"
    | "server_deleted"
    | "provider_added"
    | "provider_removed"
    | "settings_changed"
  action: string
  target?: string
  status?: string
  timestamp: string
  details?: string
  jobId?: number
}

const entries = ref<AuditEntry[]>([])
const loading = ref(true)
const error = ref("")

// Fetch all jobs as audit entries (for now, jobs are our main audit source)
const fetchAuditLog = async () => {
  loading.value = true
  error.value = ""
  try {
    const res = await fetch("/api/jobs")
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(body.error || "Failed to fetch activity log")
    }
    const jobs: Job[] = await res.json()

    // Convert jobs to audit entries
    entries.value = jobs.map((job) => ({
      id: `job-${job.id}`,
      type: "job" as const,
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
    provision_server: "Server provisioning",
    configure_server: "Server configuration",
    deploy_site: "Site deployment",
  }
  return actions[kind] || kind
}

const statusVariant = (status?: string): "success" | "warning" | "danger" | "default" => {
  if (!status) return "default"
  if (status === "succeeded") return "success"
  if (status === "failed" || status === "cancelled" || status === "timed_out") return "danger"
  if (status === "running" || status === "preparing" || status === "queued") return "warning"
  return "default"
}

const statusBadgeClass = (status?: string): string => {
  const variant = statusVariant(status)
  const mapping: Record<typeof variant, string> = {
    default: "border-border/60 bg-muted/60 text-foreground",
    success: "border-primary/30 bg-primary/10 text-primary",
    warning: "border-accent/30 bg-accent/10 text-accent",
    danger: "border-destructive/30 bg-destructive/10 text-destructive",
  }
  return mapping[variant]
}

const statusIconClass = (status?: string): string => {
  const variant = statusVariant(status)
  const mapping: Record<typeof variant, string> = {
    success: "bg-primary/10 text-primary",
    warning: "bg-accent/10 text-accent",
    danger: "bg-destructive/10 text-destructive",
    default: "bg-muted/50 text-muted-foreground",
  }
  return mapping[variant]
}

const typeIcon = (type: AuditEntry["type"]): string => {
  const icons: Record<string, string> = {
    job: "M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z",
    server_created: "M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2",
    server_deleted: "M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16",
    provider_added: "M12 4v16m8-8H4",
    provider_removed: "M20 12H4",
    settings_changed: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35",
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

    if (diffMins < 1) return "Just now"
    if (diffMins < 60) return `${diffMins}m ago`
    if (diffHours < 24) return `${diffHours}h ago`
    if (diffDays < 7) return `${diffDays}d ago`

    return date.toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
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
      <p class="text-sm text-muted-foreground">
        Complete history of all actions performed in your account.
      </p>
      <Button
        variant="link"
        size="sm"
        :disabled="loading"
        :class="cn('h-auto p-0 text-xs text-accent hover:text-accent/80 hover:no-underline')"
        @click="fetchAuditLog"
      >
        Refresh
      </Button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-8">
      <Spinner :class="cn('h-5 w-5 text-muted-foreground')" />
    </div>

    <!-- Error -->
    <Alert v-else-if="error" :class="cn('border-destructive/30 bg-destructive/10')">
      <div class="space-y-2">
        <AlertDescription :class="cn('text-sm text-destructive')">
          {{ error }}
        </AlertDescription>
        <Button
          variant="link"
          size="sm"
          :class="cn('h-auto p-0 text-xs text-destructive hover:text-destructive/80 hover:no-underline')"
          @click="fetchAuditLog"
        >
          Try again
        </Button>
      </div>
    </Alert>

    <!-- Empty state -->
    <Empty
      v-else-if="entries.length === 0"
      :class="cn('border border-dashed border-border/50 px-4 py-8 text-center')"
    >
      <EmptyMedia :class="cn('text-muted-foreground')">
        <svg class="h-10 w-10" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </EmptyMedia>
      <EmptyHeader :class="cn('gap-1')">
        <EmptyTitle :class="cn('text-sm font-medium text-foreground')">
          No activity yet
        </EmptyTitle>
        <EmptyDescription :class="cn('text-sm text-muted-foreground')">
          Actions like server provisioning, site deployments, and configuration changes will appear here.
        </EmptyDescription>
      </EmptyHeader>
    </Empty>

    <!-- Activity list -->
    <div v-else class="space-y-2">
      <Item
        v-for="entry in entries"
        :key="entry.id"
        :class="cn('items-start gap-3 rounded-lg border border-border/60 bg-card/40 px-4 py-3')"
      >
        <ItemMedia
          :class="cn('h-8 w-8 rounded-lg', statusIconClass(entry.status))"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" :d="typeIcon(entry.type)" />
          </svg>
        </ItemMedia>

        <ItemContent :class="cn('min-w-0')">
          <ItemTitle :class="cn('text-foreground')">
            <span>{{ entry.action }}</span>
            <Badge
              v-if="entry.status"
              variant="outline"
              :class="cn('text-xs', statusBadgeClass(entry.status))"
            >
              {{ entry.status }}
            </Badge>
          </ItemTitle>
          <p class="mt-0.5 text-xs text-muted-foreground">
            <span v-if="entry.target">{{ entry.target }} · </span>
            {{ formatDate(entry.timestamp) }}
          </p>
          <p v-if="entry.details" class="mt-1 text-xs text-destructive/80">
            {{ entry.details }}
          </p>
        </ItemContent>

        <ItemActions v-if="entry.jobId" :class="cn('shrink-0')">
          <Button
            as-child
            variant="link"
            size="sm"
            :class="cn('h-auto p-0 text-xs text-accent hover:text-accent/80 hover:no-underline')"
          >
            <NuxtLink :to="`/jobs/${entry.jobId}`">
              Details
            </NuxtLink>
          </Button>
        </ItemActions>
      </Item>
    </div>

    <!-- Info note -->
    <Alert :class="cn('border-border/40 bg-card/20')">
      <svg class="mt-0.5 h-4 w-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <AlertDescription :class="cn('text-xs text-muted-foreground')">
        This audit log is permanent and cannot be deleted. All actions are recorded for compliance and troubleshooting purposes.
      </AlertDescription>
    </Alert>
  </div>
</template>
