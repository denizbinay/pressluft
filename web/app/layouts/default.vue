<script setup lang="ts">
const mobileMenuOpen = ref(false)

const navLinks = [
  { label: 'Dashboard', to: '/' },
  { label: 'Sites', to: '/sites' },
  { label: 'Servers', to: '/servers' },
  { label: 'Settings', to: '/settings' },
  { label: 'Components', to: '/components' },
]

const closeMobileMenu = () => {
  mobileMenuOpen.value = false
}
</script>

<template>
  <div class="min-h-screen flex flex-col bg-surface-950 text-surface-100">
    <!-- Top Navigation -->
    <header
      class="sticky top-0 z-50 border-b border-surface-800/60 bg-surface-950/80 backdrop-blur-xl"
    >
      <nav class="mx-auto flex h-14 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <!-- Logo -->
        <NuxtLink to="/" class="flex items-center gap-2.5 group" @click="closeMobileMenu">
          <div
            class="flex h-8 w-8 items-center justify-center rounded-lg bg-accent-500/15 text-accent-400 transition-colors group-hover:bg-accent-500/25"
          >
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
          <span class="text-lg font-semibold tracking-tight text-surface-50">pressluft</span>
        </NuxtLink>

        <!-- Desktop Nav -->
        <div class="hidden items-center gap-1 md:flex">
          <NuxtLink
            v-for="link in navLinks"
            :key="link.to"
            :to="link.to"
            class="rounded-md px-3 py-1.5 text-sm font-medium text-surface-400 transition-colors hover:bg-surface-800/60 hover:text-surface-100"
            active-class="!text-surface-50 !bg-surface-800/80"
          >
            {{ link.label }}
          </NuxtLink>
        </div>

        <!-- Right side -->
        <div class="flex items-center gap-3">
          <!-- Health dot (placeholder) -->
          <div class="hidden items-center gap-2 sm:flex">
            <span class="relative flex h-2 w-2">
              <span
                class="absolute inline-flex h-full w-full animate-ping rounded-full bg-success-400 opacity-75"
              />
              <span class="relative inline-flex h-2 w-2 rounded-full bg-success-500" />
            </span>
            <span class="text-xs font-medium text-surface-400">Healthy</span>
          </div>

          <!-- Mobile hamburger -->
          <button
            class="inline-flex items-center justify-center rounded-md p-2 text-surface-400 hover:bg-surface-800/60 hover:text-surface-100 md:hidden"
            aria-label="Toggle navigation menu"
            @click="mobileMenuOpen = !mobileMenuOpen"
          >
            <svg
              v-if="!mobileMenuOpen"
              xmlns="http://www.w3.org/2000/svg"
              class="h-5 w-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h16" />
            </svg>
            <svg
              v-else
              xmlns="http://www.w3.org/2000/svg"
              class="h-5 w-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </nav>

      <!-- Mobile menu -->
      <Transition
        enter-active-class="transition duration-200 ease-out"
        enter-from-class="opacity-0 -translate-y-1"
        enter-to-class="opacity-100 translate-y-0"
        leave-active-class="transition duration-150 ease-in"
        leave-from-class="opacity-100 translate-y-0"
        leave-to-class="opacity-0 -translate-y-1"
      >
        <div
          v-if="mobileMenuOpen"
          class="border-b border-surface-800/60 bg-surface-950/95 backdrop-blur-xl md:hidden"
        >
          <div class="space-y-1 px-4 py-3">
            <NuxtLink
              v-for="link in navLinks"
              :key="link.to"
              :to="link.to"
              class="block rounded-md px-3 py-2 text-sm font-medium text-surface-400 transition-colors hover:bg-surface-800/60 hover:text-surface-100"
              active-class="!text-surface-50 !bg-surface-800/80"
              @click="closeMobileMenu"
            >
              {{ link.label }}
            </NuxtLink>
          </div>
        </div>
      </Transition>
    </header>

    <!-- Main content -->
    <main class="flex-1">
      <div class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <slot />
      </div>
    </main>

    <!-- Footer -->
    <footer class="border-t border-surface-800/40">
      <div
        class="mx-auto flex h-12 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8"
      >
        <span class="text-xs font-medium tracking-wide text-surface-500">pressluft</span>
        <span class="text-xs text-surface-600">&copy; {{ new Date().getFullYear() }}</span>
      </div>
    </footer>
  </div>
</template>
