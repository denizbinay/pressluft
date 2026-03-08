export function useApiClient() {
  const config = useRuntimeConfig()

  const normalizePath = (path: string) => (path.startsWith('/') ? path : `/${path}`)

  const apiPath = (path: string) => `${config.public.apiBase}${normalizePath(path)}`

  const apiFetch = async <T>(
    path: string,
    options: Parameters<typeof $fetch<T>>[1] = {},
  ) => {
    const requestHeaders = import.meta.server ? useRequestHeaders(['cookie']) : undefined
    return await $fetch<T>(normalizePath(path), {
      baseURL: config.public.apiBase,
      credentials: 'include',
      headers: requestHeaders,
      ...options,
    })
  }

  return {
    apiFetch,
    apiPath,
  }
}
