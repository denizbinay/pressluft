import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref, nextTick } from 'vue'

// Mock Nuxt auto-imports before importing the composable
const mockApiFetch = vi.fn()

vi.mock('vue', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue')>()
  return { ...actual }
})

// Mock useState to return simple refs
const stateStore = new Map<string, ReturnType<typeof ref>>()
vi.stubGlobal('useState', (key: string, init?: () => unknown) => {
  if (!stateStore.has(key)) {
    stateStore.set(key, ref(init ? init() : undefined))
  }
  return stateStore.get(key)!
})

// Mock useApiClient
vi.stubGlobal('useApiClient', () => ({
  apiFetch: mockApiFetch,
}))

// Now import the composable (must happen after mocks)
const { useAuth } = await import('~/composables/useAuth')

beforeEach(() => {
  stateStore.clear()
  mockApiFetch.mockReset()
})

describe('useAuth', () => {
  describe('initial state', () => {
    it('starts with null user and not initialized', () => {
      const { user, initialized, isAuthenticated } = useAuth()
      expect(user.value).toBeNull()
      expect(initialized.value).toBe(false)
      expect(isAuthenticated.value).toBe(false)
    })
  })

  describe('login', () => {
    it('sets user on successful login', async () => {
      const mockActor = {
        id: '1',
        type: 'operator',
        email: 'admin@example.com',
        role: 'admin',
        authenticated: true,
        capabilities: ['manage_servers'],
      }
      mockApiFetch.mockResolvedValue(mockActor)

      const { user, initialized, isAuthenticated, login } = useAuth()

      const actor = await login('admin@example.com', 'password123')

      expect(mockApiFetch).toHaveBeenCalledWith('/auth/login', {
        method: 'POST',
        body: { email: 'admin@example.com', password: 'password123' },
      })
      expect(actor.email).toBe('admin@example.com')
      expect(user.value).toEqual(mockActor)
      expect(initialized.value).toBe(true)
      await nextTick()
      expect(isAuthenticated.value).toBe(true)
    })

    it('propagates errors from apiFetch', async () => {
      mockApiFetch.mockRejectedValue(new Error('Invalid credentials'))

      const { login } = useAuth()

      await expect(login('bad@example.com', 'wrong')).rejects.toThrow(
        'Invalid credentials',
      )
    })
  })

  describe('logout', () => {
    it('clears user on logout', async () => {
      const mockActor = {
        id: '1',
        type: 'operator',
        email: 'admin@example.com',
        role: 'admin',
        authenticated: true,
      }
      mockApiFetch.mockResolvedValueOnce(mockActor)

      const { user, initialized, isAuthenticated, login, logout } = useAuth()

      await login('admin@example.com', 'password123')
      expect(user.value).not.toBeNull()

      mockApiFetch.mockResolvedValueOnce(undefined)
      await logout()

      expect(user.value).toBeNull()
      expect(initialized.value).toBe(true)
      await nextTick()
      expect(isAuthenticated.value).toBe(false)
      expect(mockApiFetch).toHaveBeenCalledWith('/auth/logout', {
        method: 'POST',
      })
    })
  })

  describe('fetchMe', () => {
    it('sets user when API returns actor', async () => {
      const mockActor = {
        id: '1',
        type: 'operator',
        email: 'admin@example.com',
        role: 'admin',
        authenticated: true,
      }
      mockApiFetch.mockResolvedValue(mockActor)

      const { user, initialized, fetchMe } = useAuth()

      const actor = await fetchMe()

      expect(mockApiFetch).toHaveBeenCalledWith('/auth/me')
      expect(actor).not.toBeNull()
      expect(actor!.email).toBe('admin@example.com')
      expect(user.value).toEqual(mockActor)
      expect(initialized.value).toBe(true)
    })

    it('sets user to null when API fails', async () => {
      mockApiFetch.mockRejectedValue(new Error('Unauthorized'))

      const { user, initialized, fetchMe } = useAuth()

      const actor = await fetchMe()

      expect(actor).toBeNull()
      expect(user.value).toBeNull()
      expect(initialized.value).toBe(true)
    })
  })

  describe('isAuthenticated', () => {
    it('reflects authentication state', async () => {
      const mockActor = {
        id: '1',
        type: 'operator',
        email: 'admin@example.com',
        role: 'admin',
        authenticated: false,
      }
      mockApiFetch.mockResolvedValue(mockActor)

      const { isAuthenticated, login } = useAuth()
      await login('admin@example.com', 'password123')
      await nextTick()

      // authenticated is false in the mock actor
      expect(isAuthenticated.value).toBe(false)
    })
  })
})
