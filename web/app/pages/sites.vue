<script setup lang="ts">
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
// Sites page placeholder kept for future implementation.

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

// Placeholder state until site workflows land.
const sites = ref<Site[]>([])
const loading = ref(false)
</script>

<template>
  <div class="space-y-6">
    <!-- Page header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-semibold text-foreground">Sites</h1>
        <p class="mt-2 text-base text-muted-foreground">
          This route is reserved for future site workflows.
        </p>
      </div>
      <Button
        size="sm"
        class="h-8 px-3 text-xs rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-60"
        disabled
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
        </svg>
        Not Available Yet
      </Button>
    </div>

    <!-- Planned surfaces -->
    <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <!-- Planned site setup -->
      <div class="rounded-xl border border-border/60 bg-card/30 p-6">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-accent/10">
          <svg class="h-6 w-6 text-accent" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
          </svg>
        </div>
        <h3 class="mt-5 text-base font-medium text-foreground">Planned: Site Setup</h3>
        <p class="mt-2 text-sm text-muted-foreground">
          Reserved for guided WordPress site creation once the workflow is implemented.
        </p>
      </div>

      <!-- Planned cloning -->
      <div class="rounded-xl border border-border/60 bg-card/30 p-6">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10">
          <svg class="h-6 w-6 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
          </svg>
        </div>
        <h3 class="mt-5 text-base font-medium text-foreground">Planned: Clone Site</h3>
        <p class="mt-2 text-sm text-muted-foreground">
          Reserved for staging, handoff, and duplication flows in a later round.
        </p>
      </div>

      <!-- Planned staging -->
      <div class="rounded-xl border border-border/60 bg-card/30 p-6">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-secondary/60">
          <svg class="h-6 w-6 text-secondary-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
          </svg>
        </div>
        <h3 class="mt-5 text-base font-medium text-foreground">Planned: Staging</h3>
        <p class="mt-2 text-sm text-muted-foreground">
          Reserved for future preview and sync workflows.
        </p>
      </div>

      <!-- Planned backups -->
      <div class="rounded-xl border border-border/60 bg-card/30 p-6">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10">
          <svg class="h-6 w-6 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
          </svg>
        </div>
        <h3 class="mt-5 text-base font-medium text-foreground">Planned: Backups</h3>
        <p class="mt-2 text-sm text-muted-foreground">
          Reserved for future backup and restore controls.
        </p>
      </div>

      <!-- Planned domain controls -->
      <div class="rounded-xl border border-border/60 bg-card/30 p-6">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-secondary/60">
          <svg class="h-6 w-6 text-secondary-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
          </svg>
        </div>
        <h3 class="mt-5 text-base font-medium text-foreground">Planned: Domains</h3>
        <p class="mt-2 text-sm text-muted-foreground">
          Reserved for domain, TLS, and DNS configuration once site features land.
        </p>
      </div>

      <!-- Planned monitoring -->
      <div class="rounded-xl border border-border/60 bg-card/30 p-6">
        <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-destructive/10">
          <svg class="h-6 w-6 text-destructive" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
        </div>
        <h3 class="mt-5 text-base font-medium text-foreground">Planned: Monitoring</h3>
        <p class="mt-2 text-sm text-muted-foreground">
          Reserved for uptime, performance, and alerting views in a later round.
        </p>
      </div>
    </div>

    <!-- Empty state -->
    <Card class="rounded-xl border border-border/60 bg-card/50 backdrop-blur-sm py-0 shadow-none">
      <CardContent class="px-6 py-5">
        <div class="flex flex-col items-center justify-center py-16 text-center">
          <div class="flex h-20 w-20 items-center justify-center rounded-full bg-muted/50">
            <svg class="h-10 w-10 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
            </svg>
          </div>
          <h3 class="mt-6 text-xl font-medium text-foreground">No sites yet</h3>
          <p class="mt-3 max-w-sm text-base text-muted-foreground">
            Site inventory is not implemented on this branch yet.
            For now, use the active server and provider workflows under settings.
          </p>
          <div class="mt-8 flex gap-3">
            <NuxtLink to="/servers">
              <Button
                variant="ghost"
                size="sm"
                class="h-8 px-3 text-xs rounded-lg text-foreground hover:bg-muted/50"
              >
                View Servers
              </Button>
            </NuxtLink>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Planning note -->
    <div class="rounded-xl border border-accent/30 bg-accent/5 px-6 py-5">
      <div class="flex items-start gap-4">
        <svg class="mt-0.5 h-6 w-6 shrink-0 text-accent" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <div>
          <p class="text-base font-medium text-accent">Planned surface</p>
          <p class="mt-1 text-sm text-accent/80">
            This page stays in the codebase as a placeholder for the next round. It is intentionally hidden from the primary navigation until site workflows are real.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
