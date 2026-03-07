export default defineNuxtRouteMiddleware(async (to) => {
  const { initialized, isAuthenticated, fetchMe } = useAuth()

  if (!initialized.value) {
    await fetchMe()
  }

  if (to.path === '/login') {
    if (isAuthenticated.value) {
      return navigateTo('/')
    }
    return
  }

  if (!isAuthenticated.value) {
    return navigateTo(`/login?redirect=${encodeURIComponent(to.fullPath)}`)
  }
})
