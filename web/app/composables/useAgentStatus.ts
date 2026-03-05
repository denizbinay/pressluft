import { ref, readonly, onUnmounted, onMounted, type Ref } from 'vue'
import type { AgentInfo, AgentStatusType } from './useServers'

interface UseAgentStatusOptions {
  /** Polling interval in milliseconds. Default: 15000 (15s) */
  pollInterval?: number
  /** Auto-start polling on mount. Default: true */
  autoStart?: boolean
}

/**
 * Composable for fetching and polling agent status for a single server.
 */
export function useAgentStatus(serverId: Ref<number | null>, options: UseAgentStatusOptions = {}) {
  const { pollInterval = 15000, autoStart = true } = options

  const agentInfo = ref<AgentInfo | null>(null)
  const loading = ref(false)
  const error = ref('')
  let pollTimer: ReturnType<typeof setInterval> | null = null

  const fetch = async () => {
    if (!serverId.value) return
    if (import.meta.server) return // Don't fetch on server

    loading.value = true
    error.value = ''
    try {
      const res = await globalThis.fetch(`/api/servers/${serverId.value}/agent-status`)
      if (!res.ok) throw new Error('Failed to fetch agent status')
      agentInfo.value = await res.json()
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  const startPolling = () => {
    if (import.meta.server) return // Don't poll on server
    stopPolling()
    fetch()
    pollTimer = setInterval(fetch, pollInterval)
  }

  const stopPolling = () => {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  onMounted(() => {
    if (autoStart && serverId.value) {
      startPolling()
    }
  })

  onUnmounted(() => {
    stopPolling()
  })

  return {
    agentInfo: readonly(agentInfo),
    loading: readonly(loading),
    error: readonly(error),
    fetch,
    startPolling,
    stopPolling,
  }
}

/**
 * Composable for fetching agent status for all servers.
 * Useful for the server list view.
 */
export function useAllAgentStatus(options: UseAgentStatusOptions = {}) {
  const { pollInterval = 15000, autoStart = true } = options

  const agentInfoMap = ref<Record<number, AgentInfo>>({})
  const loading = ref(false)
  const error = ref('')
  let pollTimer: ReturnType<typeof setInterval> | null = null

  const fetch = async () => {
    if (import.meta.server) return // Don't fetch on server
    loading.value = true
    error.value = ''
    try {
      const res = await globalThis.fetch('/api/servers/agents')
      if (!res.ok) throw new Error('Failed to fetch agent status')
      agentInfoMap.value = await res.json()
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  const getStatus = (serverId: number): AgentInfo | null => {
    return agentInfoMap.value[serverId] || null
  }

  const isConnected = (serverId: number): boolean => {
    const info = agentInfoMap.value[serverId]
    return info?.connected ?? false
  }

  const getStatusType = (serverId: number): AgentStatusType => {
    const info = agentInfoMap.value[serverId]
    return info?.status ?? 'unknown'
  }

  const startPolling = () => {
    if (import.meta.server) return // Don't poll on server
    stopPolling()
    fetch()
    pollTimer = setInterval(fetch, pollInterval)
  }

  const stopPolling = () => {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  onMounted(() => {
    if (autoStart) {
      startPolling()
    }
  })

  onUnmounted(() => {
    stopPolling()
  })

  return {
    agentInfoMap: readonly(agentInfoMap),
    loading: readonly(loading),
    error: readonly(error),
    fetch,
    getStatus,
    isConnected,
    getStatusType,
    startPolling,
    stopPolling,
  }
}
