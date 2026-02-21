export default defineNuxtRouteMiddleware(async (to) => {
  const auth = useAuthSession();

  // Force-check the session on navigation so expired cookies redirect deterministically.
  const forceCheck = to.path !== "/login";

  if (auth.status.value === "unknown" || (forceCheck && auth.status.value === "authenticated")) {
    try {
      await auth.restoreSession({ force: forceCheck });
    } catch {
      auth.setGuest();
    }
  }

  if (to.path === "/") {
    if (auth.isAuthenticated.value) {
      return navigateTo("/app", { replace: true });
    }
    return navigateTo("/login", { replace: true });
  }

  if (to.path === "/login") {
    if (auth.isAuthenticated.value) {
      return navigateTo("/app", { replace: true });
    }
    return;
  }

  if (!auth.isAuthenticated.value) {
    return navigateTo(
      {
        path: "/login",
        query: { redirect: to.fullPath },
      },
      { replace: true },
    );
  }
});
