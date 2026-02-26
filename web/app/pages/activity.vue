<script setup lang="ts">
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
import { Switch } from "@/components/ui/switch"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectItemText,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { cn } from "@/lib/utils"
import type { Activity, ActivityFilter } from "~/composables/useActivity"
import { useActivity } from "~/composables/useActivity"
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"

const PAGE_LIMIT = 30

const {
  activities,
  loading,
  error,
  nextCursor,
  hasMore,
  listActivity,
  streamActivity,
  markAllRead,
} = useActivity()

const isLoadingMore = ref(false)
const sentinel = ref<HTMLDivElement | null>(null)
const observer = ref<IntersectionObserver | null>(null)
const stopStream = ref<null | (() => void)>(null)
const selectedCategory = ref("all")
const attentionOnly = ref(false)

const categoryOptions = [
  { label: "All", value: "all" },
  { label: "Jobs", value: "job" },
  { label: "Servers", value: "server" },
  { label: "Providers", value: "provider" },
  { label: "Sites", value: "site" },
  { label: "Account", value: "account" },
  { label: "Security", value: "security" },
]

const levelBadgeClass = (level: Activity["level"]) => {
  const mapping: Record<string, string> = {
    info: "border-border/60 bg-muted/60 text-foreground",
    success: "border-primary/30 bg-primary/10 text-primary",
    warning: "border-accent/30 bg-accent/10 text-accent",
    error: "border-destructive/30 bg-destructive/10 text-destructive",
  }
  return mapping[level] || mapping.info
}

const levelIconClass = (level: Activity["level"]) => {
  const mapping: Record<string, string> = {
    info: "bg-muted/50 text-muted-foreground",
    success: "bg-primary/10 text-primary",
    warning: "bg-accent/10 text-accent",
    error: "bg-destructive/10 text-destructive",
  }
  return mapping[level] || mapping.info
}

const categoryIcon = (category: Activity["category"]) => {
  const icons: Record<string, string> = {
    job: "M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z",
    server: "M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2",
    provider: "M12 4v16m8-8H4",
    site: "M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3",
    account: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066",
    security: "M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2z",
  }
  return icons[category] || icons.job
}

const resourceMeta = (activity: Activity) => {
  if (activity.resource_type && activity.resource_id) {
    return `${activity.resource_type} #${activity.resource_id}`
  }
  if (activity.parent_resource_type && activity.parent_resource_id) {
    return `${activity.parent_resource_type} #${activity.parent_resource_id}`
  }
  return ""
}

const resourceLink = (activity: Activity) => {
  if (activity.resource_type === "job" && activity.resource_id) {
    return `/jobs/${activity.resource_id}`
  }
  if (activity.resource_type === "server" && activity.resource_id) {
    return `/servers/${activity.resource_id}`
  }
  return ""
}

const matchesFilter = (activity: Activity) => {
  if (selectedCategory.value !== "all" && activity.category !== selectedCategory.value) {
    return false
  }
  if (attentionOnly.value && !activity.requires_attention) {
    return false
  }
  return true
}

const currentFilter = (): ActivityFilter => {
  const filter: ActivityFilter = {}
  if (selectedCategory.value !== "all") {
    filter.category = selectedCategory.value
  }
  if (attentionOnly.value) {
    filter.requiresAttention = true
  }
  return filter
}

const formattedDate = (iso: string): string => {
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

const loadInitial = async () => {
  try {
    await listActivity(currentFilter(), { limit: PAGE_LIMIT })
  } catch {
    // Errors are surfaced via the error ref
  }
}

const loadMore = async () => {
  if (!hasMore.value || isLoadingMore.value) return
  isLoadingMore.value = true
  try {
    await listActivity(currentFilter(), {
      limit: PAGE_LIMIT,
      cursor: nextCursor.value,
      append: true,
    })
  } catch {
    // Errors are surfaced via the error ref
  } finally {
    isLoadingMore.value = false
  }
}

const setupObserver = () => {
  if (!sentinel.value) return
  observer.value?.disconnect()
  observer.value = new IntersectionObserver(
    (entries) => {
      if (entries.some((entry) => entry.isIntersecting)) {
        loadMore()
      }
    },
    { rootMargin: "200px" },
  )
  observer.value.observe(sentinel.value)
}

const handleRefresh = async () => {
  await loadInitial()
}

const handleMarkAllRead = async () => {
  await markAllRead(currentFilter())
}

const activityCountLabel = computed(() => {
  if (loading.value && activities.value.length === 0) return "Loading..."
  if (activities.value.length === 0) return "No activity yet"
  return `${activities.value.length} events`
})

onMounted(async () => {
  await loadInitial()
  stopStream.value = streamActivity({
    sinceId: activities.value[0]?.id ?? 0,
    filter: matchesFilter,
  })
  await nextTick()
  setupObserver()
})

watch([selectedCategory, attentionOnly], async () => {
  stopStream.value?.()
  await loadInitial()
  stopStream.value = streamActivity({
    sinceId: activities.value[0]?.id ?? 0,
    filter: matchesFilter,
  })
  await nextTick()
  setupObserver()
})

onBeforeUnmount(() => {
  observer.value?.disconnect()
  stopStream.value?.()
})
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-3xl font-semibold text-foreground">Activity</h1>
        <p class="mt-2 text-base text-muted-foreground">
          Account-wide audit log and infrastructure events.
        </p>
      </div>
      <div class="flex items-center gap-2">
        <span class="text-xs text-muted-foreground">{{ activityCountLabel }}</span>
        <Button
          variant="ghost"
          size="sm"
          :class="cn('h-8 rounded-lg text-xs text-muted-foreground hover:text-foreground')"
          @click="handleRefresh"
        >
          Refresh
        </Button>
        <Button
          variant="ghost"
          size="sm"
          :class="cn('h-8 rounded-lg text-xs text-muted-foreground hover:text-foreground')"
          @click="handleMarkAllRead"
        >
          Mark all read
        </Button>
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-3">
      <Select v-model="selectedCategory">
        <SelectTrigger class="w-[180px] rounded-lg border border-border/60 bg-background/60 text-xs">
          <SelectValue placeholder="Category" />
        </SelectTrigger>
        <SelectContent class="border-border/60 bg-popover text-popover-foreground">
          <SelectItem
            v-for="option in categoryOptions"
            :key="option.value"
            :value="option.value"
            class="text-foreground data-[highlighted]:bg-muted/60 data-[highlighted]:text-foreground"
          >
            <SelectItemText>{{ option.label }}</SelectItemText>
          </SelectItem>
        </SelectContent>
      </Select>

      <div class="flex items-center gap-2 rounded-lg border border-border/60 bg-background/60 px-3 py-2">
        <Switch v-model:checked="attentionOnly" />
        <span class="text-xs text-muted-foreground">Attention only</span>
      </div>
    </div>

    <Alert v-if="error" :class="cn('border-destructive/30 bg-destructive/10')">
      <div class="space-y-2">
        <AlertDescription :class="cn('text-sm text-destructive')">
          {{ error }}
        </AlertDescription>
        <Button
          variant="link"
          size="sm"
          :class="cn('h-auto p-0 text-xs text-destructive hover:text-destructive/80 hover:no-underline')"
          @click="handleRefresh"
        >
          Try again
        </Button>
      </div>
    </Alert>

    <div v-else-if="loading && activities.length === 0" class="flex items-center justify-center py-8">
      <Spinner :class="cn('h-5 w-5 text-muted-foreground')" />
    </div>

    <Empty
      v-else-if="activities.length === 0"
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
          Actions like server provisioning, deployments, and configuration changes will appear here.
        </EmptyDescription>
      </EmptyHeader>
    </Empty>

    <div v-else class="space-y-2">
      <Item
        v-for="activity in activities"
        :key="activity.id"
        :class="cn('items-start gap-3 rounded-lg border border-border/60 bg-card/40 px-4 py-3')"
      >
        <ItemMedia
          :class="cn('h-9 w-9 rounded-lg', levelIconClass(activity.level))"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" :d="categoryIcon(activity.category)" />
          </svg>
        </ItemMedia>

        <ItemContent :class="cn('min-w-0')">
          <ItemTitle :class="cn('text-foreground')">
            <span>{{ activity.title }}</span>
            <span v-if="!activity.read_at" class="ml-2 inline-flex h-1.5 w-1.5 rounded-full bg-accent" aria-hidden="true" />
            <Badge
              variant="outline"
              :class="cn('text-xs', levelBadgeClass(activity.level))"
            >
              {{ activity.level }}
            </Badge>
            <Badge
              variant="outline"
              :class="cn('text-xs border-border/60 bg-muted/40 text-muted-foreground')"
            >
              {{ activity.category }}
            </Badge>
            <Badge
              v-if="activity.requires_attention"
              variant="outline"
              :class="cn('text-xs border-destructive/30 bg-destructive/10 text-destructive')"
            >
              Needs attention
            </Badge>
          </ItemTitle>
          <p class="mt-0.5 text-xs text-muted-foreground">
            <span v-if="resourceMeta(activity)">{{ resourceMeta(activity) }} · </span>
            {{ formattedDate(activity.created_at) }}
          </p>
          <p v-if="activity.message" class="mt-1 text-xs text-muted-foreground">
            {{ activity.message }}
          </p>
        </ItemContent>

        <ItemActions v-if="resourceLink(activity)" :class="cn('shrink-0')">
          <Button
            as-child
            variant="link"
            size="sm"
            :class="cn('h-auto p-0 text-xs text-accent hover:text-accent/80 hover:no-underline')"
          >
            <NuxtLink :to="resourceLink(activity)">View Details</NuxtLink>
          </Button>
        </ItemActions>
      </Item>
    </div>

    <div v-if="isLoadingMore" class="flex items-center justify-center py-4">
      <Spinner :class="cn('h-4 w-4 text-muted-foreground')" />
    </div>

    <div v-if="hasMore" ref="sentinel" class="h-6" />
  </div>
</template>
