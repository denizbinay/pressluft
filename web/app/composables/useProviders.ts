import { ref, readonly } from 'vue'

export interface ProviderType {
  type: string
  name: string
  docs_url: string
}

export interface StoredProvider {
  id: number
  type: string
  name: string
  status: string
  created_at: string
  updated_at: string
}

export interface ValidationResult {
  valid: boolean
  read_write: boolean
  message: string
  project_name?: string
}

export function useProviders() {
  const providers = ref<StoredProvider[]>([])
  const providerTypes = ref<ProviderType[]>([])
  const loading = ref(false)
  const error = ref('')

  const fetchProviders = async () => {
    loading.value = true
    error.value = ''
    try {
      const res = await fetch('/api/providers')
      if (!res.ok) throw new Error(`Failed to fetch providers: ${res.statusText}`)
      providers.value = await res.json()
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  const fetchProviderTypes = async () => {
    try {
      const res = await fetch('/api/providers/types')
      if (!res.ok) throw new Error(`Failed to fetch provider types: ${res.statusText}`)
      providerTypes.value = await res.json()
    } catch (e: any) {
      error.value = e.message
    }
  }

  const validateToken = async (type_: string, apiToken: string): Promise<ValidationResult> => {
    const res = await fetch('/api/providers/validate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ type: type_, api_token: apiToken }),
    })
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(body.error || 'Validation request failed')
    }
    return res.json()
  }

  const createProvider = async (type_: string, name: string, apiToken: string) => {
    const res = await fetch('/api/providers', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ type: type_, name, api_token: apiToken }),
    })
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(body.error || 'Failed to create provider')
    }
    return res.json()
  }

  const deleteProvider = async (id: number) => {
    const res = await fetch(`/api/providers/${id}`, { method: 'DELETE' })
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(body.error || 'Failed to delete provider')
    }
  }

  return {
    providers: readonly(providers),
    providerTypes: readonly(providerTypes),
    loading: readonly(loading),
    error: readonly(error),
    fetchProviders,
    fetchProviderTypes,
    validateToken,
    createProvider,
    deleteProvider,
  }
}
