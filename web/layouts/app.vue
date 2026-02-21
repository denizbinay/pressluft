<template>
  <div class="app-shell">
    <header class="app-header">
      <div class="brand">
        <p class="eyebrow">Pressluft</p>
        <h1 class="title">Dashboard</h1>
      </div>

      <nav class="nav">
        <NuxtLink class="nav-link" to="/app">Overview</NuxtLink>
        <NuxtLink class="nav-link" to="/app/sites">Sites</NuxtLink>
        <NuxtLink class="nav-link" to="/app/jobs">Jobs</NuxtLink>
      </nav>

      <div class="metrics" data-testid="metrics">
        <div class="metric">
          <span class="metric-label">Jobs</span>
          <span class="metric-value mono">
            {{ metrics ? `${metrics.jobs_running} running / ${metrics.jobs_queued} queued` : "-" }}
          </span>
        </div>
        <div class="metric">
          <span class="metric-label">Nodes</span>
          <span class="metric-value mono">{{ metrics ? metrics.nodes_active : "-" }}</span>
        </div>
        <div class="metric">
          <span class="metric-label">Sites</span>
          <span class="metric-value mono">{{ metrics ? metrics.sites_total : "-" }}</span>
        </div>
      </div>

      <button class="signout" type="button" :disabled="isSigningOut" @click="signOut">
        {{ isSigningOut ? "Signing out..." : "Sign Out" }}
      </button>
    </header>

    <main class="app-main">
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
import type { MetricsResponse } from "~/lib/api/types";
import { ApiClientError } from "~/lib/api/client";

const api = useApiClient();
const auth = useAuthSession();
const isSigningOut = ref(false);

const metrics = ref<MetricsResponse | null>(null);
let metricsTimer: ReturnType<typeof setInterval> | null = null;

const METRICS_REFRESH_MS = import.meta.env.MODE === "test" ? 0 : 10_000;

onBeforeUnmount(() => {
  if (metricsTimer) {
    clearInterval(metricsTimer);
    metricsTimer = null;
  }
});

const loadMetrics = async (): Promise<void> => {
  try {
    metrics.value = await api.getMetrics();
  } catch (error) {
    if (error instanceof ApiClientError && error.status === 401) {
      auth.setGuest();
      await navigateTo("/login", { replace: true });
    }
  }
};

onMounted(() => {
  void loadMetrics();
  if (METRICS_REFRESH_MS > 0) {
    metricsTimer = setInterval(() => {
      void loadMetrics();
    }, METRICS_REFRESH_MS);
  }
});

const signOut = async (): Promise<void> => {
  isSigningOut.value = true;
  try {
    await auth.logout();
    await navigateTo("/login", { replace: true });
  } finally {
    isSigningOut.value = false;
  }
};
</script>

<style scoped>
.app-shell {
  min-height: 100dvh;
  background:
    radial-gradient(circle at top left, #dcfce7 0%, rgba(220, 252, 231, 0) 45%),
    radial-gradient(circle at right, #dbeafe 0%, rgba(219, 234, 254, 0) 40%),
    #f8fafc;
  color: #0f172a;
}

.app-header {
  display: grid;
  grid-template-columns: 1fr auto auto;
  align-items: center;
  gap: 1rem;
  padding: 1.25rem clamp(1rem, 3vw, 2rem);
  border-bottom: 1px solid rgba(226, 232, 240, 0.9);
  background: rgba(248, 250, 252, 0.65);
  backdrop-filter: blur(10px);
}

.brand {
  min-width: 0;
}

.eyebrow {
  margin: 0;
  font-size: 0.75rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #475569;
}

.title {
  margin: 0.35rem 0 0;
  font-size: 1.25rem;
}

.nav {
  display: flex;
  gap: 0.75rem;
  align-items: center;
}

.metrics {
  display: flex;
  gap: 1rem;
  align-items: center;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.metric {
  display: grid;
  gap: 0.1rem;
  padding: 0.45rem 0.65rem;
  border-radius: 0.85rem;
  border: 1px solid rgba(226, 232, 240, 0.9);
  background: rgba(255, 255, 255, 0.7);
}

.metric-label {
  font-size: 0.7rem;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: #475569;
}

.metric-value {
  font-size: 0.9rem;
  font-weight: 800;
  color: #0f172a;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
}

.nav-link {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.55rem 0.75rem;
  border-radius: 999px;
  border: 1px solid rgba(226, 232, 240, 0.9);
  background: rgba(255, 255, 255, 0.7);
  color: #0f172a;
  text-decoration: none;
  font-weight: 600;
  font-size: 0.95rem;
}

.nav-link.router-link-active {
  border-color: rgba(15, 118, 110, 0.35);
  background: rgba(240, 253, 250, 0.9);
  color: #0f766e;
}

.signout {
  border: 0;
  border-radius: 0.75rem;
  padding: 0.7rem 0.95rem;
  font-size: 0.95rem;
  font-weight: 700;
  color: #ffffff;
  background: #0f766e;
  cursor: pointer;
}

.signout:disabled {
  opacity: 0.7;
  cursor: wait;
}

.app-main {
  padding: clamp(1rem, 3vw, 2rem);
}

@media (max-width: 800px) {
  .app-header {
    grid-template-columns: 1fr;
    justify-items: start;
  }

  .nav {
    flex-wrap: wrap;
  }

  .metrics {
    justify-content: flex-start;
  }
}
</style>
