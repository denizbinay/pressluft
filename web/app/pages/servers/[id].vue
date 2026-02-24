<script setup lang="ts">
import { useServers, type StoredServer } from '~/composables/useServers'
import { useJobs, type Job } from '~/composables/useJobs'

interface ServerSection {
  key: string
  label: string
  icon: string
  description: string
}

const sections: ServerSection[] = [
  { key: 'overview', label: 'Overview', icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6', description: 'Server status and quick actions' },
  { key: 'sites', label: 'Sites', icon: 'M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9', description: 'Websites hosted on this server' },
  { key: 'settings', label: 'Settings', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z', description: 'Server configuration and management' },
  { key: 'activity', label: 'Activity', icon: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z', description: 'Recent events and job history' },
]

const route = useRoute()
const router = useRouter()

const serverId = computed(() => {
  const id = Number(route.params.id)
  return Number.isNaN(id) ? null : id
})

const { fetchServer } = useServers()

const server = ref<StoredServer | null>(null)
const loading = ref(true)
const error = ref('')

const activeSection = computed(() => {
  const tab = route.query.tab as string
  const isValid = sections.some((s) => s.key === tab)
  return isValid ? tab : 'overview'
})

const currentSection = computed(() =>
  sections.find((s) => s.key === activeSection.value)!,
)

const navigateTo = (key: string) => {
  router.push({ query: { tab: key } })
}

const isMobileSidebarOpen = ref(false)

const toggleMobileSidebar = () => {
  isMobileSidebarOpen.value = !isMobileSidebarOpen.value
}

const selectSection = (key: string) => {
  navigateTo(key)
  isMobileSidebarOpen.value = false
}

const statusVariant = (status: string): 'success' | 'warning' | 'danger' | 'default' => {
  if (status === 'ready') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'provisioning' || status === 'pending') return 'warning'
  return 'default'
}

const formatDate = (iso: string): string => {
  try {
    return new Date(iso).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return iso
  }
}

onMounted(async () => {
  if (!serverId.value) {
    error.value = 'Invalid server ID'
    loading.value = false
    return
  }

  try {
    server.value = await fetchServer(serverId.value)
  } catch (e: any) {
    error.value = e.message || 'Failed to load server'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <!-- Loading state -->
    <div v-if="loading" class="flex items-center justify-center py-20">
      <svg class="h-6 w-6 animate-spin text-surface-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
      </svg>
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="space-y-4">
      <div>
        <h1 class="text-2xl font-semibold text-surface-50">Server Not Found</h1>
        <p class="mt-1 text-sm text-surface-400">{{ error }}</p>
      </div>
      <NuxtLink
        to="/servers"
        class="inline-flex items-center gap-1 text-sm text-accent-400 transition-colors hover:text-accent-300"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
        </svg>
        Back to Servers
      </NuxtLink>
    </div>

    <!-- Server detail -->
    <template v-else-if="server">
      <!-- Page header -->
      <div class="mb-6">
        <div class="flex items-center gap-2 text-sm text-surface-500">
          <NuxtLink to="/servers" class="hover:text-surface-300 transition-colors">Servers</NuxtLink>
          <span>/</span>
          <span class="text-surface-300">{{ server.name }}</span>
        </div>
        <div class="mt-2 flex items-center gap-3">
          <h1 class="text-2xl font-semibold text-surface-50">{{ server.name }}</h1>
          <UiBadge :variant="statusVariant(server.status)">{{ server.status }}</UiBadge>
        </div>
        <p class="mt-1 text-sm text-surface-400">
          {{ server.location }} · {{ server.server_type }} · {{ server.profile_key }}
        </p>
      </div>

      <!-- Mobile section selector -->
      <div class="mb-4 lg:hidden">
        <button
          class="flex w-full items-center justify-between rounded-lg border border-surface-800/60 bg-surface-900/50 px-4 py-3 text-sm font-medium text-surface-200 transition-colors hover:bg-surface-900/70"
          @click="toggleMobileSidebar"
        >
          <span class="flex items-center gap-2.5">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              class="h-4 w-4 text-surface-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                :d="currentSection.icon"
              />
            </svg>
            {{ currentSection.label }}
          </span>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="h-4 w-4 text-surface-500 transition-transform"
            :class="{ 'rotate-180': isMobileSidebarOpen }"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        <!-- Mobile dropdown -->
        <Transition
          enter-active-class="transition duration-150 ease-out"
          enter-from-class="opacity-0 -translate-y-1"
          enter-to-class="opacity-100 translate-y-0"
          leave-active-class="transition duration-100 ease-in"
          leave-from-class="opacity-100 translate-y-0"
          leave-to-class="opacity-0 -translate-y-1"
        >
          <div
            v-if="isMobileSidebarOpen"
            class="mt-1 overflow-hidden rounded-lg border border-surface-800/60 bg-surface-900/80 backdrop-blur-sm"
          >
            <nav aria-label="Server sections">
              <button
                v-for="section in sections"
                :key="section.key"
                :class="[
                  'flex w-full items-center gap-2.5 px-4 py-2.5 text-left text-sm transition-colors',
                  activeSection === section.key
                    ? 'bg-accent-500/10 text-accent-400'
                    : 'text-surface-400 hover:bg-surface-800/50 hover:text-surface-200',
                ]"
                @click="selectSection(section.key)"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-4 w-4 shrink-0"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" :d="section.icon" />
                </svg>
                {{ section.label }}
              </button>
            </nav>
          </div>
        </Transition>
      </div>

      <!-- Desktop layout: sidebar + content -->
      <div class="flex gap-6">
        <!-- Sidebar (desktop) -->
        <aside class="hidden w-56 shrink-0 lg:block">
          <nav aria-label="Server sections" class="space-y-0.5">
            <button
              v-for="section in sections"
              :key="section.key"
              :class="[
                'flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left text-sm font-medium transition-colors',
                activeSection === section.key
                  ? 'bg-accent-500/10 text-accent-400'
                  : 'text-surface-400 hover:bg-surface-800/50 hover:text-surface-200',
              ]"
              @click="navigateTo(section.key)"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-4 w-4 shrink-0"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2"
              >
                <path stroke-linecap="round" stroke-linejoin="round" :d="section.icon" />
              </svg>
              {{ section.label }}
            </button>
          </nav>
        </aside>

        <!-- Content area -->
        <div class="min-w-0 flex-1">
          <UiCard>
            <template #header>
              <div>
                <h2 class="text-lg font-semibold text-surface-50">
                  {{ currentSection.label }}
                </h2>
                <p class="mt-0.5 text-sm text-surface-400">
                  {{ currentSection.description }}
                </p>
              </div>
            </template>

            <!-- Section content -->
            <div class="space-y-6">
              <!-- Overview -->
              <div v-if="activeSection === 'overview'" class="space-y-4">
                <div class="grid gap-4 sm:grid-cols-2">
                  <div class="rounded-lg border border-surface-800/60 bg-surface-950/40 px-4 py-3">
                    <p class="text-xs font-medium text-surface-500">Status</p>
                    <div class="mt-1 flex items-center gap-2">
                      <span
                        class="h-2 w-2 rounded-full"
                        :class="{
                          'bg-success-500': server.status === 'ready',
                          'bg-danger-500': server.status === 'failed',
                          'bg-warning-500 animate-pulse': server.status === 'provisioning' || server.status === 'pending',
                          'bg-surface-500': !['ready', 'failed', 'provisioning', 'pending'].includes(server.status),
                        }"
                      />
                      <span class="text-sm font-medium text-surface-200 capitalize">{{ server.status }}</span>
                    </div>
                  </div>
                  <div class="rounded-lg border border-surface-800/60 bg-surface-950/40 px-4 py-3">
                    <p class="text-xs font-medium text-surface-500">Provider</p>
                    <p class="mt-1 text-sm font-medium text-surface-200">{{ server.provider_type }}</p>
                  </div>
                  <div class="rounded-lg border border-surface-800/60 bg-surface-950/40 px-4 py-3">
                    <p class="text-xs font-medium text-surface-500">Location</p>
                    <p class="mt-1 text-sm font-medium text-surface-200">{{ server.location }}</p>
                  </div>
                  <div class="rounded-lg border border-surface-800/60 bg-surface-950/40 px-4 py-3">
                    <p class="text-xs font-medium text-surface-500">Server Type</p>
                    <p class="mt-1 text-sm font-medium text-surface-200">{{ server.server_type }}</p>
                  </div>
                  <div class="rounded-lg border border-surface-800/60 bg-surface-950/40 px-4 py-3">
                    <p class="text-xs font-medium text-surface-500">Profile</p>
                    <p class="mt-1 text-sm font-medium text-surface-200">{{ server.profile_key }}</p>
                  </div>
                  <div class="rounded-lg border border-surface-800/60 bg-surface-950/40 px-4 py-3">
                    <p class="text-xs font-medium text-surface-500">Created</p>
                    <p class="mt-1 text-sm font-medium text-surface-200">{{ formatDate(server.created_at) }}</p>
                  </div>
                </div>

                <!-- Provider Server ID (if available) -->
                <div v-if="server.provider_server_id" class="rounded-lg border border-surface-800/60 bg-surface-950/40 px-4 py-3">
                  <p class="text-xs font-medium text-surface-500">Provider Server ID</p>
                  <p class="mt-1 font-mono text-sm text-surface-200">{{ server.provider_server_id }}</p>
                </div>

                <!-- Quick actions placeholder -->
                <div class="rounded-lg border border-dashed border-surface-700/50 px-4 py-6 text-center">
                  <p class="text-sm text-surface-500">
                    Quick actions (reboot, stop, start, SSH access) will be available here.
                  </p>
                </div>
              </div>

              <!-- Sites -->
              <div v-if="activeSection === 'sites'" class="space-y-4">
                <div class="rounded-lg border border-dashed border-surface-700/50 px-4 py-8 text-center">
                  <h3 class="text-sm font-medium text-surface-200">No sites yet</h3>
                  <p class="mt-1 text-sm text-surface-500">
                    WordPress sites deployed to this server will appear here.
                  </p>
                  <p class="mt-3 text-xs text-surface-600">
                    Site management features are coming soon.
                  </p>
                </div>
              </div>

              <!-- Settings -->
              <div v-if="activeSection === 'settings'" class="space-y-4">
                <div class="rounded-lg border border-dashed border-surface-700/50 px-4 py-8 text-center">
                  <p class="text-sm text-surface-500">
                    Server configuration options (SSH keys, firewall rules, backups, monitoring) will be available here.
                  </p>
                </div>
              </div>

              <!-- Activity -->
              <div v-if="activeSection === 'activity'" class="space-y-4">
                <ServerActivity :server-id="server.id" />
              </div>
            </div>
          </UiCard>
        </div>
      </div>
    </template>
  </div>
</template>
