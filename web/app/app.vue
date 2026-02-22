<template>
  <main class="dashboard">
    <header class="dashboard__header">
      <h1>Pressluft</h1>
      <p>Nuxt UI and Go API baseline.</p>
      <p class="dashboard__origin">Current origin: {{ currentOrigin }}</p>
      <div class="health-badge" :data-state="healthState">
        <strong>Backend:</strong>
        <span>{{ healthLabel }}</span>
      </div>
    </header>

    <section class="dashboard__panel">
      <h2>Status</h2>
      <ul>
        <li>Nuxt project is initialized via CLI</li>
        <li>Generated assets are copied into Go embed path</li>
        <li>Go serves dashboard and API from one process</li>
        <li>Live API check: <code>/api/health</code></li>
      </ul>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

type HealthState = 'checking' | 'healthy' | 'unreachable'

const healthState = ref<HealthState>('checking')
const currentOrigin = ref('unknown')
let healthTimer: ReturnType<typeof setInterval> | undefined

const healthLabel = computed(() => {
  if (healthState.value === 'healthy') {
    return 'healthy'
  }
  if (healthState.value === 'unreachable') {
    return 'unreachable'
  }
  return 'checking...'
})

async function checkHealth(): Promise<void> {
  try {
    const response = await fetch('/api/health', {
      method: 'GET',
      headers: { Accept: 'application/json' },
    })

    if (!response.ok) {
      healthState.value = 'unreachable'
      return
    }

    const payload = (await response.json()) as { status?: string }
    healthState.value = payload.status === 'healthy' ? 'healthy' : 'unreachable'
  } catch {
    healthState.value = 'unreachable'
  }
}

onMounted(() => {
  currentOrigin.value = window.location.origin
  void checkHealth()
  healthTimer = setInterval(() => {
    if (healthState.value === 'healthy') {
      return
    }
    void checkHealth()
  }, 2000)
})

onBeforeUnmount(() => {
  if (healthTimer) {
    clearInterval(healthTimer)
  }
})
</script>

<style scoped>
:global(html),
:global(body),
:global(#__nuxt) {
  margin: 0;
  min-height: 100%;
  background: #0b1220;
}

.dashboard {
  --bg: #0b1220;
  --bg-elevated: #111a2c;
  --panel-bg: #16233aee;
  --text: #e8edf7;
  --text-muted: #adc0de;
  --border: #2d3e5f;
  --badge-checking-bg: #3a3018;
  --badge-checking-border: #d1a94c;
  --badge-healthy-bg: #173329;
  --badge-healthy-border: #48aa78;
  --badge-unreachable-bg: #3c1e26;
  --badge-unreachable-border: #d36b82;
  min-height: 100vh;
  padding: 2rem;
  color: var(--text);
  font-family: Manrope, "Segoe UI", sans-serif;
  background:
    radial-gradient(circle at 80% 0%, #1a2f4f 0%, transparent 40%),
    linear-gradient(160deg, var(--bg) 10%, var(--bg-elevated) 100%);
}

.dashboard__header h1 {
  margin: 0;
  font-size: 2rem;
}

.dashboard__header p {
  margin-top: 0.5rem;
}

.dashboard__origin {
  color: var(--text-muted);
}

.health-badge {
  display: inline-flex;
  gap: 0.5rem;
  margin-top: 0.75rem;
  padding: 0.45rem 0.7rem;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: color-mix(in oklab, var(--bg-elevated), black 10%);
}

.health-badge[data-state='checking'] {
  border-color: var(--badge-checking-border);
  background: var(--badge-checking-bg);
}

.health-badge[data-state='healthy'] {
  border-color: var(--badge-healthy-border);
  background: var(--badge-healthy-bg);
}

.health-badge[data-state='unreachable'] {
  border-color: var(--badge-unreachable-border);
  background: var(--badge-unreachable-bg);
}

.dashboard__panel {
  max-width: 42rem;
  margin-top: 1.5rem;
  padding: 1rem 1.25rem;
  border: 1px solid var(--border);
  border-radius: 0.75rem;
  background: var(--panel-bg);
}

.dashboard__panel h2 {
  margin-top: 0;
}
</style>
