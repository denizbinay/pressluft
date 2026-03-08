import { ref, readonly } from 'vue'
import type { ProviderType, StoredProvider, ValidationResult } from '~/lib/api-contract'
export type { ProviderType, StoredProvider, ValidationResult } from '~/lib/api-contract'

export function useProviders() {
  const { apiFetch } = useApiClient()
  const providers = ref<StoredProvider[]>([])
  const providerTypes = ref<ProviderType[]>([])
  const loading = ref(false)
  const error = ref('')

  const fetchProviders = async () => {
    loading.value = true
    error.value = ''
    try {
      providers.value = await apiFetch<StoredProvider[]>('/providers')
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  const fetchProviderTypes = async () => {
    try {
      providerTypes.value = await apiFetch<ProviderType[]>('/providers/types')
    } catch (e: any) {
      error.value = e.message
    }
  }

  const validateToken = async (type_: string, apiToken: string): Promise<ValidationResult> => {
    return await apiFetch<ValidationResult>('/providers/validate', {
      method: 'POST',
      body: { type: type_, api_token: apiToken },
    })
  }

  const createProvider = async (type_: string, name: string, apiToken: string) => {
    return await apiFetch('/providers', {
      method: 'POST',
      body: { type: type_, name, api_token: apiToken },
    })
  }

  const deleteProvider = async (id: number) => {
    await apiFetch(`/providers/${id}`, { method: 'DELETE' })
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
