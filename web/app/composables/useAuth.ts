import { computed } from 'vue'
import type { AuthActor } from '~/lib/api-contract'
import { parseAuthActor } from '~/lib/api-runtime'

export function useAuth() {
  const user = useState<AuthActor | null>('auth-user', () => null)
  const initialized = useState<boolean>('auth-initialized', () => false)
  const { apiFetch } = useApiClient()

  const fetchMe = async () => {
    try {
      const actor = parseAuthActor(await apiFetch('/auth/me'))
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
    const actor = parseAuthActor(await apiFetch('/auth/login', {
      method: 'POST',
      body: { email, password },
    }))
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
