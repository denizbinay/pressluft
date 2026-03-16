<script setup lang="ts">
import type { StoredSite, SiteHealthResponse } from "~/composables/useSites";
import { formatRelativeTime } from "~/composables/useSiteHelpers";

const props = defineProps<{
  site: StoredSite;
  siteHealth: SiteHealthResponse | null;
}>();

const healthChecks = computed(() => props.siteHealth?.snapshot?.checks || []);
const healthServices = computed(() => props.siteHealth?.snapshot?.services || []);

const findCheck = (name: string) => healthChecks.value.find((c) => c.name === name);

const availabilityChecks = computed(() => [
  { label: "Website reachable", check: findCheck("home-page") },
  { label: "Login page reachable", check: findCheck("login-page") },
]);

const wordpressChecks = computed(() => [
  { label: "WordPress installed", check: findCheck("wordpress-installed") },
  { label: "Home URL matches", check: findCheck("wordpress-home-url") },
  { label: "Site URL matches", check: findCheck("wordpress-siteurl") },
]);

const serviceChecks = computed(() =>
  healthServices.value.map((service) => ({
    label: service.name,
    ok: service.active_state === "active",
    detail: service.active_state === "active" ? "Active" : service.active_state,
  })),
);

const healthScore = computed(() => {
  if (!props.siteHealth?.snapshot) return null;

  const totalChecks = healthChecks.value.length + serviceChecks.value.length;
  if (totalChecks === 0) return null;

  const passingChecks = healthChecks.value.filter((check) => check.ok).length;
  const passingServices = serviceChecks.value.filter((service) => service.ok).length;
  return Math.round(((passingChecks + passingServices) / totalChecks) * 100);
});

const redisService = computed(() => healthServices.value.find((s) => s.name === "redis-server"));
</script>

<template>
  <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)] lg:items-start">
    <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="size-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" /></svg>
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Site Health</p>
        </div>
        <span v-if="healthScore !== null" class="rounded-full border px-2.5 py-1 text-xs font-semibold" :class="healthScore >= 80 ? 'border-primary/30 bg-primary/10 text-primary' : healthScore >= 50 ? 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-200' : 'border-destructive/30 bg-destructive/10 text-destructive'">{{ healthScore }}% health</span>
      </div>

        <div class="mt-5">
          <p class="text-xs font-medium text-muted-foreground">Availability</p>
          <div class="mt-2 space-y-1.5">
            <div v-for="item in availabilityChecks" :key="item.label" class="flex items-center justify-between gap-3 text-sm">
              <span class="text-foreground">{{ item.label }}</span>
              <span v-if="item.check" class="flex items-center gap-1.5" :class="item.check.ok ? 'text-primary' : 'text-destructive'">
                <svg v-if="item.check.ok" xmlns="http://www.w3.org/2000/svg" class="size-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" class="size-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" /></svg>
                {{ item.check.ok ? "OK" : (item.check.detail || "Issue") }}
              </span>
              <span v-else class="text-muted-foreground">--</span>
            </div>
          </div>
        </div>

        <div class="mt-4">
          <p class="text-xs font-medium text-muted-foreground">WordPress</p>
          <div class="mt-2 space-y-1.5">
            <div v-for="item in wordpressChecks" :key="item.label" class="flex items-center justify-between gap-3 text-sm">
              <span class="text-foreground">{{ item.label }}</span>
              <span v-if="item.check" class="flex items-center gap-1.5" :class="item.check.ok ? 'text-primary' : 'text-destructive'">
                <svg v-if="item.check.ok" xmlns="http://www.w3.org/2000/svg" class="size-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" class="size-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" /></svg>
                {{ item.check.ok ? "OK" : (item.check.detail || "Issue") }}
              </span>
              <span v-else class="text-muted-foreground">--</span>
            </div>
          </div>
        </div>

        <div class="mt-4">
          <p class="text-xs font-medium text-muted-foreground">Runtime</p>
          <div class="mt-2 space-y-1.5">
            <div class="flex items-center justify-between gap-3 text-sm">
              <span class="text-foreground">PHP version</span>
              <span class="text-muted-foreground">{{ site.php_version || "--" }}</span>
            </div>
            <div class="flex items-center justify-between gap-3 text-sm">
              <span class="text-foreground">WordPress core</span>
              <span class="text-muted-foreground">{{ site.wordpress_version || "--" }}</span>
            </div>
            <div class="flex items-center justify-between gap-3 text-sm">
              <span class="text-foreground">Primary hostname</span>
              <span class="truncate text-muted-foreground">{{ site.primary_domain || "--" }}</span>
            </div>
          </div>
        </div>

        <div v-if="serviceChecks.length" class="mt-4">
          <p class="text-xs font-medium text-muted-foreground">Services</p>
          <div class="mt-2 space-y-1.5">
            <div v-for="service in serviceChecks" :key="service.label" class="flex items-center justify-between gap-3 text-sm">
              <span class="text-foreground">{{ service.label }}</span>
              <span :class="service.ok ? 'text-primary' : 'text-destructive'">{{ service.detail }}</span>
            </div>
          </div>
        </div>

        <div v-if="!siteHealth?.snapshot && !siteHealth?.agent_connected" class="mt-4 rounded-xl border border-border/40 bg-muted/10 px-3 py-2.5 text-xs text-muted-foreground">
          Health checks require an active agent connection.
        </div>
        <div v-else-if="siteHealth?.agent_connected && !siteHealth?.snapshot" class="mt-4 rounded-xl border border-border/40 bg-muted/10 px-3 py-2.5 text-xs text-muted-foreground">
          Agent connected. Waiting for health snapshot...
        </div>

        <p v-if="siteHealth?.last_health_check_at" class="mt-3 text-xs text-muted-foreground">
          Checked {{ formatRelativeTime(siteHealth.last_health_check_at) }}
        </p>
      </div>

    <div class="space-y-4">
      <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
        <div class="flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="size-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" /></svg>
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Traffic</p>
        </div>
        <div class="mt-4 space-y-3">
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Requests (24h)</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Bandwidth</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Avg response time</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Server errors (5xx)</span>
            <span class="text-muted-foreground">--</span>
          </div>
        </div>
      </div>

      <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
        <div class="flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="size-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" /></svg>
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Performance</p>
        </div>
        <div class="mt-4 space-y-3">
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Page cache</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Redis</span>
            <span v-if="redisService" :class="redisService.active_state === 'active' ? 'text-primary' : 'text-destructive'">
              {{ redisService.active_state === "active" ? "Active" : redisService.active_state }}
            </span>
            <span v-else class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">CDN</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">PHP workers</span>
            <span class="text-muted-foreground">--</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Avg response time</span>
            <span class="text-muted-foreground">--</span>
          </div>
        </div>
      </div>

      <div class="rounded-2xl border border-border/60 bg-background/70 p-5">
        <div class="flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="size-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" /></svg>
          <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">Security</p>
        </div>
        <div class="mt-4 space-y-3">
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">SSL certificate</span>
            <span class="text-muted-foreground">Pending check</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Security scans</span>
            <span class="text-muted-foreground">Not configured</span>
          </div>
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="text-foreground">Backups</span>
            <span class="text-muted-foreground">Managed elsewhere</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
