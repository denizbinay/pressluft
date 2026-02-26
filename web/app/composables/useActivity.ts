import { computed, ref } from "vue"

export interface Activity {
  id: number
  event_type: string
  category: string
  level: "info" | "success" | "warning" | "error" | string
  resource_type?: string
  resource_id?: number
  parent_resource_type?: string
  parent_resource_id?: number
  actor_type: string
  actor_id?: string
  title: string
  message?: string
  payload?: string
  requires_attention: boolean
  read_at?: string
  created_at: string
}

export interface ActivityListResponse {
  data: Activity[]
  next_cursor?: string
}

export interface ActivityFilter {
  category?: string
  resourceType?: string
  resourceId?: number
  parentResourceType?: string
  parentResourceId?: number
  requiresAttention?: boolean
  unreadOnly?: boolean
}

export interface ListActivityOptions {
  cursor?: string
  limit?: number
  append?: boolean
}

export interface ActivityStreamOptions {
  sinceId?: number
  filter?: (activity: Activity) => boolean
  onEvent?: (activity: Activity) => void
  onError?: (event: Event) => void
  maxItems?: number
}

const buildActivityParams = (
  filter: ActivityFilter = {},
  options: ListActivityOptions = {},
) => {
  const params = new URLSearchParams()

  if (options.cursor) params.set("cursor", options.cursor)
  if (options.limit) params.set("limit", String(options.limit))

  if (filter.category) params.set("category", filter.category)
  if (filter.resourceType) params.set("resource_type", filter.resourceType)
  if (filter.resourceId) params.set("resource_id", String(filter.resourceId))
  if (filter.parentResourceType) {
    params.set("parent_resource_type", filter.parentResourceType)
  }
  if (filter.parentResourceId) {
    params.set("parent_resource_id", String(filter.parentResourceId))
  }
  if (typeof filter.requiresAttention === "boolean") {
    params.set("requires_attention", filter.requiresAttention ? "true" : "false")
  }
  if (filter.unreadOnly) params.set("unread_only", "true")

  return params
}

export function useActivity() {
  const activities = ref<Activity[]>([])
  const loading = ref(false)
  const error = ref("")
  const nextCursor = ref("")
  const unreadCount = ref(0)

  const hasMore = computed(() => Boolean(nextCursor.value))

  const upsertActivity = (activity: Activity, prepend = true) => {
    const index = activities.value.findIndex((item) => item.id === activity.id)
    if (index >= 0) {
      activities.value.splice(index, 1, activity)
      return
    }

    activities.value = prepend
      ? [activity, ...activities.value]
      : [...activities.value, activity]
  }

  const listActivity = async (
    filter: ActivityFilter = {},
    options: ListActivityOptions = {},
  ) => {
    loading.value = true
    error.value = ""
    try {
      const params = buildActivityParams(filter, options)
      const query = params.toString()
      const res = await fetch(`/api/activity${query ? `?${query}` : ""}`)
      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: res.statusText }))
        throw new Error(body.error || "Failed to fetch activity")
      }
      const payload = (await res.json()) as ActivityListResponse
      const next = payload.next_cursor || ""

      if (options.append) {
        activities.value = [...activities.value, ...payload.data]
      } else {
        activities.value = payload.data
      }
      nextCursor.value = next
      return payload
    } catch (e: any) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  const listServerActivity = async (
    serverId: number,
    options: ListActivityOptions = {},
  ) => {
    loading.value = true
    error.value = ""
    try {
      const params = buildActivityParams({}, options)
      const query = params.toString()
      const res = await fetch(
        `/api/servers/${serverId}/activity${query ? `?${query}` : ""}`,
      )
      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: res.statusText }))
        throw new Error(body.error || "Failed to fetch server activity")
      }
      const payload = (await res.json()) as ActivityListResponse
      const next = payload.next_cursor || ""

      if (options.append) {
        activities.value = [...activities.value, ...payload.data]
      } else {
        activities.value = payload.data
      }
      nextCursor.value = next
      return payload
    } catch (e: any) {
      error.value = e.message
      throw e
    } finally {
      loading.value = false
    }
  }

  const streamActivity = (options: ActivityStreamOptions = {}) => {
    const initialSinceId = options.sinceId ?? 0
    let lastSeenId = initialSinceId
    let stream: EventSource | null = null
    let closed = false
    let retryCount = 0
    let retryTimeout: ReturnType<typeof setTimeout> | null = null

    const scheduleReconnect = () => {
      if (closed || retryTimeout) return
      const delay = Math.min(30000, 1000 * Math.pow(2, retryCount))
      retryCount += 1
      retryTimeout = setTimeout(() => {
        retryTimeout = null
        connect()
      }, delay)
    }

    const connect = () => {
      if (closed) return
      try {
        const query = lastSeenId > 0 ? `?since_id=${lastSeenId}` : ""
        stream = new EventSource(`/api/activity/stream${query}`)

        stream.addEventListener("activity", (evt) => {
          try {
            const parsed = JSON.parse((evt as MessageEvent).data) as Activity
            if (parsed.id > lastSeenId) {
              lastSeenId = parsed.id
            }
            retryCount = 0
            if (options.filter && !options.filter(parsed)) return
            upsertActivity(parsed, true)
            if (options.maxItems && activities.value.length > options.maxItems) {
              activities.value = activities.value.slice(0, options.maxItems)
            }
            options.onEvent?.(parsed)
          } catch {
            // Ignore malformed activity payloads
          }
        })

        stream.onerror = (event) => {
          options.onError?.(event)
          if (stream) {
            stream.close()
            stream = null
          }
          scheduleReconnect()
        }
      } catch {
        scheduleReconnect()
      }
    }

    connect()

    return () => {
      closed = true
      if (stream) {
        stream.close()
        stream = null
      }
      if (retryTimeout) {
        clearTimeout(retryTimeout)
        retryTimeout = null
      }
    }
  }

  const fetchUnreadCount = async (filter: ActivityFilter = {}) => {
    try {
      const params = buildActivityParams(filter)
      const query = params.toString()
      const res = await fetch(
        `/api/activity/unread-count${query ? `?${query}` : ""}`,
      )
      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: res.statusText }))
        throw new Error(body.error || "Failed to fetch unread count")
      }
      const payload = (await res.json()) as { count: number }
      unreadCount.value = payload.count
      return payload.count
    } catch (e: any) {
      error.value = e.message
      throw e
    }
  }

  const markRead = async (activityId: number) => {
    const res = await fetch(`/api/activity/${activityId}/read`, {
      method: "POST",
    })
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(body.error || "Failed to mark activity read")
    }
    const payload = (await res.json()) as Activity
    const previous = activities.value.find((item) => item.id === activityId)
    const wasUnread = previous && !previous.read_at
    upsertActivity(payload, false)
    if (wasUnread && unreadCount.value > 0) {
      unreadCount.value -= 1
    }
    return payload
  }

  const markAllRead = async (filter: ActivityFilter = {}) => {
    const params = buildActivityParams(filter)
    const query = params.toString()
    const res = await fetch(
      `/api/activity/read-all${query ? `?${query}` : ""}`,
      { method: "POST" },
    )
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(body.error || "Failed to mark activity read")
    }
    const now = new Date().toISOString()
    const matchesFilter = (activity: Activity) => {
      if (filter.category && activity.category !== filter.category) return false
      if (filter.resourceType && activity.resource_type !== filter.resourceType) return false
      if (filter.resourceId && activity.resource_id !== filter.resourceId) return false
      if (filter.parentResourceType && activity.parent_resource_type !== filter.parentResourceType) {
        return false
      }
      if (filter.parentResourceId && activity.parent_resource_id !== filter.parentResourceId) {
        return false
      }
      if (typeof filter.requiresAttention === "boolean") {
        if (activity.requires_attention !== filter.requiresAttention) return false
      }
      if (filter.unreadOnly && activity.read_at) return false
      return true
    }
    activities.value = activities.value.map((activity) => {
      if (!activity.read_at && matchesFilter(activity)) {
        return { ...activity, read_at: now }
      }
      return activity
    })
    await fetchUnreadCount(filter)
  }

  return {
    activities,
    loading,
    error,
    nextCursor,
    hasMore,
    unreadCount,
    listActivity,
    listServerActivity,
    streamActivity,
    fetchUnreadCount,
    markRead,
    markAllRead,
  }
}
