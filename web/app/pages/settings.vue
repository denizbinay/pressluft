<script setup lang="ts">
interface SettingsSection {
  key: string
  label: string
  icon: string
  description: string
}

const sections: SettingsSection[] = [
  { key: 'general', label: 'General', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z', description: 'Application name, timezone, and general preferences' },
  { key: 'providers', label: 'Providers', icon: 'M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10', description: 'Cloud providers and infrastructure connections' },
  { key: 'servers', label: 'Servers', icon: 'M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01', description: 'Managed servers, SSH keys, and access configuration' },
  { key: 'sites', label: 'Sites', icon: 'M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9', description: 'Websites, domains, and deployment targets' },
  { key: 'notifications', label: 'Notifications', icon: 'M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9', description: 'Alerts, email digests, and webhook integrations' },
  { key: 'security', label: 'Security', icon: 'M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z', description: 'Authentication, two-factor, and session management' },
  { key: 'api-keys', label: 'API Keys', icon: 'M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z', description: 'API tokens and access credentials' },
]

const route = useRoute()
const router = useRouter()

const activeSection = computed(() => {
  const tab = route.query.tab as string
  const isValid = sections.some((s) => s.key === tab)
  return isValid ? tab : 'general'
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
</script>

<template>
  <div>
    <!-- Page header -->
    <div class="mb-6">
      <h1 class="text-2xl font-semibold text-surface-50">Settings</h1>
      <p class="mt-1 text-sm text-surface-400">
        Manage your application configuration and preferences.
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
          <nav aria-label="Settings sections">
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
        <nav aria-label="Settings sections" class="space-y-0.5">
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

          <!-- Placeholder content per section -->
          <div class="space-y-6">
            <!-- General -->
            <div v-if="activeSection === 'general'" class="space-y-4">
              <div class="rounded-lg border border-dashed border-surface-700/50 px-4 py-8 text-center">
                <p class="text-sm text-surface-500">
                  Application name, default timezone, language preferences, and theme settings will go here.
                </p>
              </div>
            </div>

            <!-- Providers -->
            <div v-if="activeSection === 'providers'" class="space-y-4">
              <div class="rounded-lg border border-dashed border-surface-700/50 px-4 py-8 text-center">
                <p class="text-sm text-surface-500">
                  Cloud provider integrations (AWS, DigitalOcean, Hetzner, etc.) and connection management will go here.
                </p>
              </div>
            </div>

            <!-- Servers -->
            <div v-if="activeSection === 'servers'" class="space-y-4">
              <div class="rounded-lg border border-dashed border-surface-700/50 px-4 py-8 text-center">
                <p class="text-sm text-surface-500">
                  Server inventory, SSH key management, and remote access configuration will go here.
                </p>
              </div>
            </div>

            <!-- Sites -->
            <div v-if="activeSection === 'sites'" class="space-y-4">
              <div class="rounded-lg border border-dashed border-surface-700/50 px-4 py-8 text-center">
                <p class="text-sm text-surface-500">
                  Website management, domain configuration, and deployment target settings will go here.
                </p>
              </div>
            </div>

            <!-- Notifications -->
            <div v-if="activeSection === 'notifications'" class="space-y-4">
              <div class="rounded-lg border border-dashed border-surface-700/50 px-4 py-8 text-center">
                <p class="text-sm text-surface-500">
                  Alert rules, email digest preferences, Slack/webhook integrations will go here.
                </p>
              </div>
            </div>

            <!-- Security -->
            <div v-if="activeSection === 'security'" class="space-y-4">
              <div class="rounded-lg border border-dashed border-surface-700/50 px-4 py-8 text-center">
                <p class="text-sm text-surface-500">
                  Password policy, two-factor authentication, session management, and audit log will go here.
                </p>
              </div>
            </div>

            <!-- API Keys -->
            <div v-if="activeSection === 'api-keys'" class="space-y-4">
              <div class="rounded-lg border border-dashed border-surface-700/50 px-4 py-8 text-center">
                <p class="text-sm text-surface-500">
                  API token generation, scopes, expiration, and revocation management will go here.
                </p>
              </div>
            </div>
          </div>
        </UiCard>
      </div>
    </div>
  </div>
</template>
