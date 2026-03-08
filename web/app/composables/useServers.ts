import { ref, readonly } from 'vue'
import type {
  AgentInfo,
  AgentStatusMapResponse,
  CreateServerRequest,
  CreateServerResponse,
  DeleteServerResponse,
  ServerCatalogResponse,
  ServerTypePrice,
  ServicesResponse,
  StoredServer,
} from '~/lib/api-contract'
export type {
  AgentInfo,
  ServerCatalogResponse,
  ServerTypePrice,
  ServicesResponse,
  StoredServer,
} from '~/lib/api-contract'
import {
  parseAgentInfo,
  parseAgentStatusMapResponse,
  parseCreateServerResponse,
  parseDeleteServerResponse,
  parseServerCatalogResponse,
  parseServicesResponse,
  parseStoredServer,
  parseStoredServers,
} from '~/lib/api-runtime'

export type AgentStatusType = AgentInfo['status']

export function useServers() {
  const { apiFetch } = useApiClient()
  const servers = ref<StoredServer[]>([])
  const profiles = ref<ServerCatalogResponse['profiles']>([])
  const catalog = ref<ServerCatalogResponse['catalog'] | null>(null)
  const loading = ref(false)
  const saving = ref(false)
  const error = ref('')

  const fetchServers = async () => {
    loading.value = true
    error.value = ''
    try {
      servers.value = parseStoredServers(await apiFetch('/servers'))
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  const fetchCatalog = async (providerId: number) => {
    error.value = ''
    catalog.value = null
    profiles.value = []
    const body = parseServerCatalogResponse(await apiFetch(`/servers/catalog?provider_id=${providerId}`))
    catalog.value = body.catalog
    profiles.value = body.profiles
  }

  const createServer = async (payload: CreateServerRequest): Promise<CreateServerResponse> => {
    saving.value = true
    error.value = ''
    try {
      return parseCreateServerResponse(await apiFetch('/servers', {
        method: 'POST',
        body: payload,
      }))
    } finally {
      saving.value = false
    }
  }

  const deleteServer = async (serverId: number): Promise<DeleteServerResponse> => {
    error.value = ''
    return parseDeleteServerResponse(await apiFetch(`/servers/${serverId}`, {
      method: 'DELETE',
    }))
  }

  const fetchServer = async (serverId: number): Promise<StoredServer> => {
    error.value = ''
    return parseStoredServer(await apiFetch(`/servers/${serverId}`))
  }

  const fetchAgentStatus = async (serverId: number): Promise<AgentInfo> => {
    return parseAgentInfo(await apiFetch(`/servers/${serverId}/agent-status`))
  }

  const fetchAllAgentStatus = async (): Promise<AgentStatusMapResponse> => {
    return parseAgentStatusMapResponse(await apiFetch('/servers/agents'))
  }

  const fetchServices = async (serverId: number): Promise<ServicesResponse> => {
    return parseServicesResponse(await apiFetch(`/servers/${serverId}/services`))
  }

  return {
    servers: readonly(servers),
    profiles: readonly(profiles),
    catalog: readonly(catalog),
    loading: readonly(loading),
    saving: readonly(saving),
    error: readonly(error),
    fetchServers,
    fetchServer,
    fetchCatalog,
    createServer,
    deleteServer,
    fetchAgentStatus,
    fetchAllAgentStatus,
    fetchServices,
  }
}
