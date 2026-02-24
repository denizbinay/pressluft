<script setup lang="ts">
const route = useRoute()
const open = ref(false)

// Navigation sections with subheadings (flat structure for UNavigationMenu)
const navSections = [
  {
    title: 'Main',
    items: [
      { label: 'Dashboard', icon: 'i-lucide-layout-dashboard', to: '/' },
    ]
  },
  {
    title: 'Resources',
    items: [
      { label: 'Sites', icon: 'i-lucide-globe', to: '/sites' },
      { label: 'Servers', icon: 'i-lucide-server', to: '/servers' },
    ]
  },
  {
    title: 'System',
    items: [
      { label: 'Settings', icon: 'i-lucide-settings', to: '/settings' },
      { label: 'Components', icon: 'i-lucide-box', to: '/components' },
    ]
  },
]

// Flatten all items for command palette search
const flatNavItems = navSections.flatMap(section => section.items)

// Command palette groups
const searchGroups = computed(() => [{
  id: 'navigation',
  label: 'Navigation',
  items: flatNavItems.map(item => ({
    id: item.label!.toLowerCase(),
    label: item.label,
    icon: item.icon,
    to: item.to
  }))
}])
</script>

<template>
  <UDashboardGroup>
    <!-- Sidebar -->
    <UDashboardSidebar
      v-model:open="open"
      collapsible
      resizable
    >
      <!-- Logo / Brand -->
      <template #header>
        <NuxtLink to="/" class="flex items-center gap-2.5 px-2 py-1.5">
          <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-cyan-500/15 text-cyan-400">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              class="h-4.5 w-4.5"
            >
              <path d="M12 2L2 7l10 5 10-5-10-5z" />
              <path d="M2 17l10 5 10-5" />
              <path d="M2 12l10 5 10-5" />
            </svg>
          </div>
          <span class="text-lg font-semibold tracking-tight text-white">pressluft</span>
        </NuxtLink>
      </template>

      <!-- Navigation -->
      <template #default="{ collapsed }">
        <div v-if="!collapsed" class="space-y-6">
          <div v-for="section in navSections" :key="section.title">
            <!-- Section subheading -->
            <p class="mb-2 px-3 text-xs font-medium uppercase tracking-wider text-neutral-500">
              {{ section.title }}
            </p>
            <!-- Navigation items for this section -->
            <UNavigationMenu
              :items="section.items"
              orientation="vertical"
              highlight
            />
          </div>
        </div>
        <!-- Collapsed: just icons, no labels -->
        <UNavigationMenu
          v-else
          :items="flatNavItems"
          orientation="vertical"
          highlight
        />
      </template>

      <!-- User Panel -->
      <template #footer="{ collapsed }">
        <UserPanel :collapsed="collapsed" />
      </template>
    </UDashboardSidebar>

    <!-- Command Palette -->
    <UDashboardSearch :groups="searchGroups" />

    <!-- Main Content Area -->
    <UDashboardPanel grow>
      <!-- Topbar -->
      <template #header>
        <UDashboardNavbar title="" align="left">
          <template #leading>
            <UDashboardSidebarCollapse />
          </template>

          <!-- Breadcrumb -->
          <div class="flex items-center gap-1 text-sm text-neutral-400">
            <span>pressluft</span>
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
            <template v-if="route.path === '/'">
              <span class="text-neutral-200">Dashboard</span>
            </template>
            <template v-else>
              <span class="text-neutral-200 capitalize">{{ route.path.replace('/', '').replace('-', ' ') }}</span>
            </template>
          </div>

          <!-- Right side utilities -->
          <template #right>
            <!-- Command palette hint -->
            <UButton
              variant="ghost"
              color="neutral"
              size="sm"
              class="hidden sm:flex"
              @click="open = true"
            >
              <template #leading>
                <UIcon name="i-lucide-search" class="h-4 w-4" />
              </template>
              <span class="text-neutral-400">Search...</span>
              <template #trailing>
                <span class="ml-2 text-xs text-neutral-500 border border-neutral-700 rounded px-1.5 py-0.5">⌘K</span>
              </template>
            </UButton>

            <!-- Mobile search button -->
            <UButton
              variant="ghost"
              color="neutral"
              icon="i-lucide-search"
              class="sm:hidden"
              @click="open = true"
            />

            <!-- Help -->
            <UButton variant="ghost" color="neutral" icon="i-lucide-help-circle" />

            <!-- Notifications -->
            <UButton variant="ghost" color="neutral" icon="i-lucide-bell" />
          </template>
        </UDashboardNavbar>
      </template>

      <!-- Page Content -->
      <template #body>
        <div class="h-full overflow-auto p-4 sm:p-6 lg:p-8">
          <div class="mx-auto max-w-7xl">
            <slot />
          </div>
        </div>
      </template>
    </UDashboardPanel>
  </UDashboardGroup>
</template>
