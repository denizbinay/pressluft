import { ref, readonly } from 'vue'

export interface ServerProfile {
  key: string
  name: string
  description: string
  artifact_path: string
}

export interface ServerLocation {
  name: string
  description: string
  country?: string
  city?: string
  network_zone?: string
}

export interface ServerTypePrice {
  location_name: string
  hourly_gross: string
  monthly_gross: string
  currency: string
}

export interface ServerTypeOption {
  name: string
  description: string
  cores: number
  memory_gb: number
  disk_gb: number
  architecture: string
  prices: ServerTypePrice[]
}

export interface ServerImageOption {
  name: string
  description: string
  type: string
  os_flavor?: string
  os_version?: string
  architecture?: string
}

export interface ServerCatalog {
  locations: ServerLocation[]
  server_types: ServerTypeOption[]
  images: ServerImageOption[]
}

export interface StoredServer {
  id: number
  provider_id: number
  provider_type: string
  provider_server_id?: string
  name: string
  location: string
  server_type: string
  image: string
  profile_key: string
  status: string
  action_id?: string
  action_status?: string
  created_at: string
  updated_at: string
}

export interface CreateServerInput {
  provider_id: number
  name: string
  location: string
  server_type: string
  image: string
  profile_key: string
}

export interface CreateServerResponse {
  server_id: number
  job_id: number
  status: string
}

export function useServers() {
  const servers = ref<StoredServer[]>([])
  const profiles = ref<ServerProfile[]>([])
  const catalog = ref<ServerCatalog | null>(null)
  const loading = ref(false)
  const saving = ref(false)
  const error = ref('')

  const fetchServers = async () => {
    loading.value = true
    error.value = ''
    try {
      const res = await fetch('/api/servers')
      if (!res.ok) throw new Error(`Failed to fetch servers: ${res.statusText}`)
      servers.value = await res.json()
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
    const res = await fetch(`/api/servers/catalog?provider_id=${providerId}`)
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(body.error || 'Failed to fetch server catalog')
    }
    const body = await res.json()
    catalog.value = body.catalog
    profiles.value = body.profiles
  }

  const createServer = async (payload: CreateServerInput): Promise<CreateServerResponse> => {
    saving.value = true
    error.value = ''
    try {
      const res = await fetch('/api/servers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: res.statusText }))
        throw new Error(body.error || 'Failed to create server')
      }
      return await res.json() as CreateServerResponse
    } finally {
      saving.value = false
    }
  }

  return {
    servers: readonly(servers),
    profiles: readonly(profiles),
    catalog: readonly(catalog),
    loading: readonly(loading),
    saving: readonly(saving),
    error: readonly(error),
    fetchServers,
    fetchCatalog,
    createServer,
  }
}
