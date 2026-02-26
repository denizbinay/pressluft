<script setup lang="ts">
import type { Activity } from "~/composables/useActivity"
import { onBeforeUnmount, onMounted, ref } from "vue"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { useActivity } from "~/composables/useActivity"

interface Props {
  serverId: number
}

const props = defineProps<Props>()

const {
  activities,
  loading,
  error,
  listServerActivity,
  streamActivity,
} = useActivity()

const stopStream = ref<null | (() => void)>(null)

type StatusVariant = "success" | "warning" | "danger" | "default"

const badgeVariantClass = (variant: StatusVariant): string => {
  const mapping: Record<StatusVariant, string> = {
    default: "border-border/60 bg-muted/60 text-foreground",
    success: "border-primary/30 bg-primary/10 text-primary",
    warning: "border-accent/30 bg-accent/10 text-accent",
    danger: "border-destructive/30 bg-destructive/10 text-destructive",
  }

  return mapping[variant]
}

const levelVariant = (level: Activity["level"]): StatusVariant => {
  if (level === "success") return "success"
  if (level === "warning") return "warning"
  if (level === "error") return "danger"
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

const serverFilter = (activity: Activity) => {
  const isResource =
    activity.resource_type === "server" && activity.resource_id === props.serverId
  const isParent =
    activity.parent_resource_type === "server" &&
    activity.parent_resource_id === props.serverId
  return isResource || isParent
}

const fetchServerActivity = async () => {
  try {
    await listServerActivity(props.serverId, { limit: 20 })
  } catch {
    // Errors are surfaced via the error ref
  }
}

const activityMeta = (activity: Activity) => {
  if (activity.resource_type && activity.resource_id) {
    if (activity.resource_type === "job") return `Job #${activity.resource_id}`
    if (activity.resource_type === "server") return `Server #${activity.resource_id}`
    return `${activity.resource_type} #${activity.resource_id}`
  }
  if (activity.parent_resource_type && activity.parent_resource_id) {
    return `${activity.parent_resource_type} #${activity.parent_resource_id}`
  }
  return ""
}

const activityLink = (activity: Activity) => {
  if (activity.resource_type === "job" && activity.resource_id) {
    return `/jobs/${activity.resource_id}`
  }
  if (activity.resource_type === "server" && activity.resource_id) {
    return `/servers/${activity.resource_id}`
  }
  return ""
}

onMounted(async () => {
  await fetchServerActivity()
  stopStream.value = streamActivity({
    sinceId: activities.value[0]?.id ?? 0,
    filter: serverFilter,
  })
})

onBeforeUnmount(() => {
  stopStream.value?.()
})
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
        @click="fetchServerActivity"
      >
        Try again
      </Button>
    </Alert>

    <div v-else-if="activities.length === 0" class="rounded-lg border border-dashed border-border/50 px-4 py-8 text-center">
      <p class="text-sm text-muted-foreground">
        No activity recorded for this server yet.
      </p>
    </div>

    <div v-else class="space-y-3">
      <div
        v-for="activity in activities"
        :key="activity.id"
        class="flex items-center justify-between gap-4 rounded-lg border border-border/60 bg-card/40 px-4 py-3"
      >
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-foreground">{{ activity.title }}</span>
            <Badge
              variant="outline"
              :class="cn(badgeVariantClass(levelVariant(activity.level)), 'text-xs')"
            >
              {{ activity.level }}
            </Badge>
            <Badge
              v-if="activity.requires_attention"
              variant="outline"
              :class="cn('text-xs border-destructive/30 bg-destructive/10 text-destructive')"
            >
              Needs attention
            </Badge>
          </div>
          <p class="mt-0.5 text-xs text-muted-foreground">
            <span v-if="activityMeta(activity)">{{ activityMeta(activity) }} · </span>
            {{ formatDate(activity.created_at) }}
          </p>
          <p v-if="activity.message" class="mt-1 text-xs text-muted-foreground">
            {{ activity.message }}
          </p>
        </div>
        <Button
          v-if="activityLink(activity)"
          as-child
          variant="link"
          size="sm"
          :class="cn('h-auto shrink-0 px-0 text-xs text-accent hover:text-accent/80')"
        >
          <NuxtLink :to="activityLink(activity)">View Details</NuxtLink>
        </Button>
      </div>
    </div>
  </div>
</template>
