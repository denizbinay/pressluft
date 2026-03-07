import { computed } from 'vue'

export interface AuthActor {
  id: string
  type: string
  email: string
  role: string
  authenticated: boolean
  auth_source?: string
}

export function useAuth() {
  const user = useState<AuthActor | null>('auth-user', () => null)
  const initialized = useState<boolean>('auth-initialized', () => false)
  const config = useRuntimeConfig()

  const apiFetch = async <T>(path: string, options: Parameters<typeof $fetch<T>>[1] = {}) => {
    const requestHeaders = import.meta.server ? useRequestHeaders(['cookie']) : undefined
    return await $fetch<T>(path, {
      baseURL: config.public.apiBase,
      credentials: 'include',
      headers: requestHeaders,
      ...options,
    })
  }

  const fetchMe = async () => {
    try {
      const actor = await apiFetch<AuthActor>('/auth/me')
      user.value = actor
      return actor
    } catch {
      user.value = null
      return null
    } finally {
      initialized.value = true
    }
  }

  const login = async (email: string, password: string) => {
    const actor = await apiFetch<AuthActor>('/auth/login', {
      method: 'POST',
      body: { email, password },
    })
    user.value = actor
    initialized.value = true
    return actor
  }

  const logout = async () => {
    await apiFetch('/auth/logout', { method: 'POST' })
    user.value = null
    initialized.value = true
  }

  return {
    user,
    initialized,
    isAuthenticated: computed(() => !!user.value?.authenticated),
    fetchMe,
    login,
    logout,
  }
}
