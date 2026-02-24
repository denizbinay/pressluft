<script setup lang="ts">
// Sites page - Top-level route for managing WordPress sites across all servers
// This will be the primary workflow for agency users

// Placeholder data structure for future implementation
interface Site {
  id: number
  name: string
  domain: string
  serverId: number
  serverName: string
  status: 'active' | 'staging' | 'maintenance' | 'error'
  wpVersion: string
  phpVersion: string
  lastBackup?: string
  createdAt: string
}

// Empty state - no sites yet
const sites = ref<Site[]>([])
const loading = ref(false)
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-semibold text-surface-50">Sites</h1>
        <p class="mt-2 text-base text-surface-400">
          Manage WordPress sites across all your servers.
        </p>
      </div>
      <UiButton size="sm" disabled>
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
        </svg>
        Create Site
      </UiButton>
    </div>

    <!-- Feature preview cards -->
    <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <!-- Create Sites -->
      <div class="rounded-xl border border-surface-800/60 bg-surface-900/30 p-6">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-accent-500/10">
          <svg class="h-6 w-6 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
          </svg>
        </div>
        <h3 class="mt-5 text-base font-medium text-surface-200">Create Sites</h3>
        <p class="mt-2 text-sm text-surface-500">
          Deploy new WordPress sites with optimized configurations for your agency workflow.
        </p>
      </div>

      <!-- Clone Sites -->
      <div class="rounded-xl border border-surface-800/60 bg-surface-900/30 p-6">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-primary-500/10">
          <svg class="h-6 w-6 text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
          </svg>
        </div>
        <h3 class="mt-5 text-base font-medium text-surface-200">Clone Sites</h3>
        <p class="mt-2 text-sm text-surface-500">
          Duplicate existing sites for staging, testing, or client handoffs with one click.
        </p>
      </div>

      <!-- Staging Environments -->
      <div class="rounded-xl border border-surface-800/60 bg-surface-900/30 p-6">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-warning-500/10">
          <svg class="h-6 w-6 text-warning-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
          </svg>
        </div>
        <h3 class="mt-5 text-base font-medium text-surface-200">Staging Environments</h3>
        <p class="mt-2 text-sm text-surface-500">
          Push changes from staging to production with confidence using built-in sync tools.
        </p>
      </div>

      <!-- Backups -->
      <div class="rounded-xl border border-surface-800/60 bg-surface-900/30 p-6">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-success-500/10">
          <svg class="h-6 w-6 text-success-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
          </svg>
        </div>
        <h3 class="mt-5 text-base font-medium text-surface-200">Automated Backups</h3>
        <p class="mt-2 text-sm text-surface-500">
          Schedule automatic backups and restore sites to any point in time.
        </p>
      </div>

      <!-- Domain Management -->
      <div class="rounded-xl border border-surface-800/60 bg-surface-900/30 p-6">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-info-500/10">
          <svg class="h-6 w-6 text-info-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
          </svg>
        </div>
        <h3 class="mt-5 text-base font-medium text-surface-200">Domain Management</h3>
        <p class="mt-2 text-sm text-surface-500">
          Configure custom domains, SSL certificates, and DNS settings for each site.
        </p>
      </div>

      <!-- Site Monitoring -->
      <div class="rounded-xl border border-surface-800/60 bg-surface-900/30 p-6">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-danger-500/10">
          <svg class="h-6 w-6 text-danger-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
        </div>
        <h3 class="mt-5 text-base font-medium text-surface-200">Site Monitoring</h3>
        <p class="mt-2 text-sm text-surface-500">
          Track uptime, performance metrics, and get alerts when issues arise.
        </p>
      </div>
    </div>

    <!-- Empty state -->
    <UiCard>
      <div class="flex flex-col items-center justify-center py-16 text-center">
        <div class="flex h-20 w-20 items-center justify-center rounded-full bg-surface-800/50">
          <svg class="h-10 w-10 text-surface-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
          </svg>
        </div>
        <h3 class="mt-6 text-xl font-medium text-surface-200">No sites yet</h3>
        <p class="mt-3 max-w-sm text-base text-surface-500">
          Sites will appear here once you deploy WordPress to your managed servers.
          First, make sure you have at least one server provisioned.
        </p>
        <div class="mt-8 flex gap-3">
          <NuxtLink to="/servers">
            <UiButton variant="ghost" size="sm">
              View Servers
            </UiButton>
          </NuxtLink>
        </div>
      </div>
    </UiCard>

    <!-- Coming soon note -->
    <div class="rounded-xl border border-accent-500/20 bg-accent-500/5 px-6 py-5">
      <div class="flex items-start gap-4">
        <svg class="mt-0.5 h-6 w-6 shrink-0 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <div>
          <p class="text-base font-medium text-accent-300">Site management coming soon</p>
          <p class="mt-1 text-sm text-accent-400/80">
            We're building the site deployment and management features. For now, focus on setting up your servers and provider connections.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
