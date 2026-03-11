<script setup lang="ts">
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Bell } from "lucide-vue-next"
import { onBeforeUnmount, onMounted, watch } from "vue"
import type { Activity as ActivityEvent } from "~/composables/useActivity"
import { useActivity } from "~/composables/useActivity"

const router = useRouter()
const notificationsOpen = ref(false)
const notificationFilter = { requiresAttention: true }

const {
  activities: notifications,
  unreadCount,
  listActivity,
  streamActivity,
  fetchUnreadCount,
  markAllRead,
  markRead,
} = useActivity()

const resolveActivityLink = (activity: ActivityEvent) => {
  if (activity.event_type === "site.deleted") {
    return "/sites"
  }
  if (activity.resource_type === "job" && activity.resource_id) {
    return `/jobs/${activity.resource_id}`
  }
  if (activity.resource_type === "server" && activity.resource_id) {
    return `/servers/${activity.resource_id}`
  }
  if (activity.resource_type === "site" && activity.resource_id) {
    return `/sites/${activity.resource_id}`
  }
  if (activity.resource_type === "domain") {
    return "/domains"
  }
  if (activity.parent_resource_type === "site" && activity.parent_resource_id) {
    return `/sites/${activity.parent_resource_id}`
  }
  if (activity.parent_resource_type === "server" && activity.parent_resource_id) {
    return `/servers/${activity.parent_resource_id}`
  }
  return "/activity"
}

const notificationMeta = (activity: ActivityEvent) => {
  const parts: string[] = []
  if (activity.resource_type && activity.resource_id) {
    parts.push(`${activity.resource_type} #${activity.resource_id}`)
  }
  if (activity.created_at) {
    try {
      parts.push(
        new Date(activity.created_at).toLocaleDateString("en-US", {
          month: "short",
          day: "numeric",
          hour: "2-digit",
          minute: "2-digit",
        }),
      )
    } catch {
      parts.push(activity.created_at)
    }
  }
  return parts.join(" · ")
}

const refreshNotifications = async () => {
  try {
    await listActivity(notificationFilter, { limit: 6 })
  } catch {
    // Errors are surfaced via the composable error state
  }
}

const refreshUnreadCount = async () => {
  try {
    await fetchUnreadCount(notificationFilter)
  } catch {
    // Errors are surfaced via the composable error state
  }
}

const handleNotificationClick = async (activity: ActivityEvent) => {
  if (!activity.read_at) {
    try {
      await markRead(activity.id)
    } catch {
      // Ignore mark read failures for navigation
    }
  }
  router.push(resolveActivityLink(activity))
  notificationsOpen.value = false
}

const handleMarkAllRead = async () => {
  try {
    await markAllRead(notificationFilter)
  } catch {
    // Errors are surfaced via the composable error state
  }
  await refreshNotifications()
}

const stopNotificationStream = ref<null | (() => void)>(null)

onMounted(async () => {
  await refreshNotifications()
  await refreshUnreadCount()
  stopNotificationStream.value = streamActivity({
    sinceId: notifications.value[0]?.id ?? "",
    filter: (activity) => activity.requires_attention,
    maxItems: 10,
    onEvent: (activity) => {
      if (!activity.read_at) {
        unreadCount.value += 1
      }
    },
  })
})

onBeforeUnmount(() => {
  stopNotificationStream.value?.()
})

watch(
  () => notificationsOpen.value,
  (open) => {
    if (open) {
      refreshNotifications()
      refreshUnreadCount()
    }
  },
)
</script>

<template>
  <DropdownMenu v-model:open="notificationsOpen">
    <DropdownMenuTrigger as-child>
      <Button
        variant="ghost"
        class="relative rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted/70 focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
        type="button"
        aria-label="Notifications"
      >
        <Bell class="h-4 w-4" aria-hidden="true" />
        <span
          v-if="unreadCount > 0"
          class="absolute -right-0.5 -top-0.5 flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-destructive px-1 text-[10px] font-semibold text-destructive-foreground"
        >
          {{ unreadCount > 9 ? "9+" : unreadCount }}
        </span>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent
      align="end"
      :side-offset="8"
      class="w-80 rounded-xl border border-border/60 bg-popover p-0 shadow-xl"
    >
      <div class="flex items-center justify-between px-4 py-3">
        <DropdownMenuLabel class="p-0 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          Notifications
        </DropdownMenuLabel>
        <Badge variant="outline" class="border-border/60 text-[10px] text-muted-foreground">
          {{ unreadCount }} unread
        </Badge>
      </div>
      <DropdownMenuSeparator class="mx-0 my-0 h-px bg-border/60" />
      <div class="max-h-80 overflow-auto">
        <div
          v-if="notifications.length === 0"
          class="px-4 py-6 text-sm text-muted-foreground"
        >
          No notifications yet.
        </div>
        <DropdownMenuItem
          v-for="activity in notifications"
          :key="activity.id"
          class="items-start gap-3 px-4 py-3"
          @select="handleNotificationClick(activity)"
        >
          <span
            class="mt-1 h-2 w-2 rounded-full"
            :class="activity.read_at ? 'bg-muted-foreground/40' : 'bg-accent'"
          />
          <div class="min-w-0 flex-1">
            <p class="text-sm font-medium text-foreground">
              {{ activity.title }}
            </p>
            <p class="mt-1 text-xs text-muted-foreground">
              {{ notificationMeta(activity) }}
            </p>
            <p v-if="activity.message" class="mt-1 text-xs text-muted-foreground">
              {{ activity.message }}
            </p>
          </div>
        </DropdownMenuItem>
      </div>
      <DropdownMenuSeparator class="mx-0 my-0 h-px bg-border/60" />
      <div class="flex items-center justify-between px-4 py-3">
        <Button
          variant="ghost"
          size="sm"
          class="h-7 px-2 text-xs text-muted-foreground hover:text-foreground"
          @click.stop="handleMarkAllRead"
        >
          Mark all read
        </Button>
        <Button
          variant="ghost"
          size="sm"
          class="h-7 px-2 text-xs text-muted-foreground hover:text-foreground"
          @click.stop="router.push('/activity')"
        >
          View all
        </Button>
      </div>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
