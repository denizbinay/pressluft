<script setup lang="ts">
import type { Job } from "~/composables/useJobs"
import { onMounted, ref } from "vue"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

interface Props {
  serverId: number
}

const props = defineProps<Props>()

const jobs = ref<Job[]>([])
const loading = ref(true)
const error = ref("")

type StatusVariant = "success" | "warning" | "danger" | "default"
type BadgeSize = "xs" | "sm" | "md" | "lg" | "xl"

const badgeVariantClass = (variant: StatusVariant): string => {
  const mapping: Record<StatusVariant, string> = {
    default: "border-border/60 bg-muted/60 text-foreground",
    success: "border-primary/30 bg-primary/10 text-primary",
    warning: "border-accent/30 bg-accent/10 text-accent",
    danger: "border-destructive/30 bg-destructive/10 text-destructive",
  }

  return mapping[variant]
}

const badgeSizeClass = (size: BadgeSize): string => {
  const mapping: Record<BadgeSize, string> = {
    xs: "px-1.5 py-0.5 text-[10px]",
    sm: "px-2 py-0.5 text-xs",
    md: "px-2.5 py-1 text-sm",
    lg: "px-3 py-1.5 text-sm",
    xl: "px-3.5 py-2 text-base",
  }

  return mapping[size]
}

const fetchServerJobs = async () => {
  loading.value = true
  error.value = ""
  try {
    const res = await fetch(`/api/servers/${props.serverId}/jobs`)
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(body.error || "Failed to fetch server jobs")
    }
    jobs.value = await res.json()
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

const statusVariant = (status: string): StatusVariant => {
  if (status === "succeeded") return "success"
  if (status === "failed" || status === "cancelled" || status === "timed_out") return "danger"
  if (status === "running" || status === "preparing" || status === "queued") return "warning"
  return "default"
}

const formatDate = (iso: string): string => {
  try {
    return new Date(iso).toLocaleDateString("en-US", {
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

const kindLabel = (kind: string): string => {
  const labels: Record<string, string> = {
    provision_server: "Server Provisioning",
    configure_server: "Server Configuration",
    deploy_site: "Site Deployment",
  }
  return labels[kind] || kind
}

const badgeClass = (variant: StatusVariant, size: BadgeSize = "sm") => {
  return cn(badgeVariantClass(variant), badgeSizeClass(size))
}

onMounted(fetchServerJobs)
</script>

<template>
  <div class="space-y-4">
    <div v-if="loading" class="flex items-center justify-center py-8">
      <svg class="h-5 w-5 animate-spin text-muted-foreground" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
    </div>

    <Alert
      v-else-if="error"
      :class="cn('border-destructive/30 bg-destructive/10')"
    >
      <AlertDescription :class="cn('text-sm text-destructive')">
        {{ error }}
      </AlertDescription>
      <Button
        variant="link"
        size="sm"
        :class="cn('mt-2 h-auto px-0 text-xs text-destructive hover:text-destructive/80')"
        @click="fetchServerJobs"
      >
        Try again
      </Button>
    </Alert>

    <div v-else-if="jobs.length === 0" class="rounded-lg border border-dashed border-border/50 px-4 py-8 text-center">
      <p class="text-sm text-muted-foreground">
        No activity recorded for this server yet.
      </p>
    </div>

    <div v-else class="space-y-3">
      <div
        v-for="job in jobs"
        :key="job.id"
        class="flex items-center justify-between rounded-lg border border-border/60 bg-card/40 px-4 py-3"
      >
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-foreground">{{ kindLabel(job.kind) }}</span>
            <Badge
              variant="outline"
              :class="badgeClass(statusVariant(job.status), 'sm')"
            >
              {{ job.status }}
            </Badge>
          </div>
          <p class="mt-0.5 text-xs text-muted-foreground">
            {{ formatDate(job.created_at) }}
            <span v-if="job.last_error" class="text-destructive"> · {{ job.last_error }}</span>
          </p>
        </div>
        <Button
          as-child
          variant="link"
          size="sm"
          :class="cn('h-auto shrink-0 px-0 text-xs text-accent hover:text-accent/80')"
        >
          <NuxtLink :to="`/jobs/${job.id}`">View Details</NuxtLink>
        </Button>
      </div>
    </div>
  </div>
</template>
